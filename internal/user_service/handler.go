package user_service

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/Gowtham1729/go_social_media/gen/user_service/v1"
	"github.com/Gowtham1729/go_social_media/internal/config"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"net/http"
	"time"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	db     *mongo.Database
	logger *zap.Logger
	cfg    *config.Config
}

func NewUserServer(db *mongo.Database, logger *zap.Logger, cfg *config.Config) *UserServer {
	return &UserServer{db: db, logger: logger, cfg: cfg}
}

func (s *UserServer) SignUpUser(ctx context.Context, req *pb.SignUpUserRequest) (*pb.SignUpUserResponse, error) {
	if err := validateSignUpRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %v", err)
	}

	collection := s.db.Collection("users")

	existingUser := &pb.User{}
	err := collection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"username": req.Username},
			{"email": req.Email},
		},
	}).Decode(existingUser)

	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "User with this username or email already exists")
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		s.logger.Error("Database error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	currentTime := time.Now().UTC()
	newUserID := uuid.NewString()

	_, err = collection.InsertOne(
		ctx,
		bson.M{
			"id":         newUserID,
			"username":   req.Username,
			"email":      req.Email,
			"name":       req.Name,
			"created_at": currentTime,
			"updated_at": currentTime,
			"password":   string(hashedPassword),
		})

	if err != nil {
		s.logger.Error("Failed to insert user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to create user")
	}

	return &pb.SignUpUserResponse{User: &pb.User{
		Id:        newUserID,
		Username:  req.Username,
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: timestamppb.New(currentTime),
		UpdatedAt: timestamppb.New(currentTime),
	}}, nil
}

func validateSignUpRequest(req *pb.SignUpUserRequest) error {
	if req.Username == "" || req.Email == "" || req.Name == "" || req.Password == "" {
		return errors.New("all fields are required")
	}

	return nil
}

type DBUser struct {
	ID        string    `bson:"id"`
	Username  string    `bson:"username"`
	Email     string    `bson:"email"`
	Name      string    `bson:"name"`
	Password  string    `bson:"password"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func (s *UserServer) SignInUser(ctx context.Context, req *pb.SignInUserRequest) (*pb.SignInUserResponse, error) {
	if err := validateSignInRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %v", err)
	}

	collection := s.db.Collection("users")

	var dbuser DBUser

	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&dbuser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(codes.NotFound, "User not found")
		}
		s.logger.Error("Database error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(dbuser.Password), []byte(req.Password)); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid credentials")
	}

	token, err := generateJWT(dbuser.ID, s.cfg.UserService.JwtSecret)
	if err != nil {
		s.logger.Error("Failed to generate JWT", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return &pb.SignInUserResponse{
		AccessToken: token,
		User: &pb.User{
			Id:        dbuser.ID,
			Username:  dbuser.Username,
			Email:     dbuser.Email,
			Name:      dbuser.Name,
			CreatedAt: timestamppb.New(dbuser.CreatedAt),
			UpdatedAt: timestamppb.New(dbuser.UpdatedAt),
		},
	}, nil

}

func generateJWT(userID, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(secret))
}

func validateSignInRequest(req *pb.SignInUserRequest) error {
	if req.Username == "" || req.Password == "" {
		return errors.New("username and password are required")
	}
	return nil
}

func connectToMongoDB(cfg *config.Config) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.Database.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func RunUserServer() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to initializa logger: %w", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", zap.Error(err))
		return fmt.Errorf("failed to load config: %w", err)
	}

	mongoClient, err := connectToMongoDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.Database.DB)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GrpcPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)
	pb.RegisterUserServiceServer(grpcServer, NewUserServer(db, logger, cfg))
	reflection.Register(grpcServer)

	// Set up gRPC-Gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf(":%d", cfg.Server.GrpcPort), opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC-Gateway: %w", err)
	}

	// Start servers
	go func() {
		logger.Info("Starting gRPC server", zap.Int("port", cfg.Server.GrpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	logger.Info("Starting HTTP server", zap.Int("port", cfg.Server.HttpPort))
	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.HttpPort), mux)

}

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		h, err := handler(ctx, req)
		logger.Info("gRPC call",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return h, err
	}
}
