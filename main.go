package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	// Import the generated protobuf code
	pb "github.com/bege13mot/consignment-service/proto/consignment"
	userProto "github.com/bege13mot/user-service/proto/auth"
	vesselProto "github.com/bege13mot/vessel-service/proto/vessel"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	defaultGrpcAddr     = "localhost:50051"
	defaultGrpcHTTPAddr = "10.0.0.65:8082"

	defaultVesselAddress = "localhost:50053"
	defaultUserAddress   = "localhost:50054"

	defaultDbHost = "localhost:27017"

	defaultConsulAddr = "localhost:8500"
)

var (
	// Get database details from environment variables
	dbHost = os.Getenv("DB_HOST")

	grpcAddr     = os.Getenv("GRPC_ADDR")
	grpcHTTPAddr = os.Getenv("GRPC_HTTP_ADDR")
	consulAddr   = os.Getenv("CONSUL_ADDR")
	vesselAddr   = os.Getenv("VESSEL_ADDR")
	userAddr     = os.Getenv("USER_ADDR")
)

func initVar() {
	if dbHost == "" {
		log.Println("Use default DB connection settings")
		dbHost = defaultDbHost
	}

	if grpcAddr == "" && grpcHTTPAddr == "" && vesselAddr == "" && userAddr == "" {
		log.Println("Use default GRPC connection settings")
		grpcAddr = defaultGrpcAddr
		vesselAddr = defaultVesselAddress
		userAddr = defaultUserAddress
		grpcHTTPAddr = defaultGrpcHTTPAddr
	}

	if consulAddr == "" {
		log.Println("Use default Consul connection settings")
		consulAddr = defaultConsulAddr
	}
}

// allowCORS allows Cross Origin Resoruce Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept", "Authorization"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	fmt.Printf("CORS preflight request for %s \n", r.URL.Path)
}

// AuthInterceptor is a high-order function which takes a HandlerFunc
// and returns a function, which takes a context, request and response interface.
// The token is extracted from the context set in our consignment-cli, that
// token is then sent over to the user service to be validated.
// If valid, the call is passed along to the handler. If not,
// an error is returned.
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	//monitoring
	grpc_prometheus.UnaryServerInterceptor(ctx, req, info, handler)

	meta, _ := metadata.FromIncomingContext(ctx)
	token := meta["authorization"]
	if len(token) != 1 || token[0] == "null" {
		return nil, errors.New("No auth meta-data found in request")
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(userAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect to User gRPC: %v, address: %v", err, userAddr)
	}
	defer conn.Close()

	c := userProto.NewAuthClient(conn)
	authResp, err := c.ValidateToken(ctx, &userProto.Token{Token: token[0]})

	if err != nil {
		log.Fatalf("Could not authenticate to userProto: %v", err)
	}
	log.Println("Auth resp:", authResp)

	return handler(ctx, req)
}

func main() {

	// Database host from the environment variables
	initVar()
	// host := os.Getenv("DB_HOST")
	// if host == "" {
	// 	host = defaultDbHost
	// }

	session, err := CreateSession(dbHost)

	// Mgo creates a 'master' session, we need to end that session
	// before the main function closes.
	defer session.Close()

	if err != nil {
		// We're wrapping the error returned from our CreateSession
		// here to add some context to the error.
		log.Panicf("Could not connect to datastore with host %s - %v", dbHost, err)
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(vesselAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Did not connect to Vessel gRPC: %v, address: %v", err, vesselAddr)
	}
	defer conn.Close()

	vesselClient := vesselProto.NewVesselServiceClient(conn)

	////////////////////
	//Connect to Consul
	config := consulapi.DefaultConfig()
	config.Address = consulAddr
	consul, err := consulapi.NewClient(config)
	if err != nil {
		log.Println("Error during connect to Consul, ", err)
	}

	serviceID := "consignment-service_" + grpcAddr

	//Register in Consul
	defer func() {
		cErr := consul.Agent().ServiceDeregister(serviceID)
		if cErr != nil {
			log.Println("Cant add service to Consul", cErr)
			return
		}
		log.Println("Deregistered in Consul", serviceID)
	}()

	err = consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "consignment-service",
		Port:    50051,
		Address: "host123",
		// Check: &consulapi.AgentServiceCheck{
		// 	CheckID:  "health_check",
		// 	Name:     "User-Service health status",
		// 	Interval: "10s",
		// 	GRPC:     "host123:50054",
		// },
	})
	if err != nil {
		log.Println("Couldn't register service in Consul, ", err)
	}
	log.Println("Registered in Consul", serviceID)

	//Test section
	health, _, err := consul.Health().Service("user-service", "", false, nil)
	if err != nil {
		log.Println("Cant get alive services")
	}

	fmt.Println("HEALTH: ", len(health))
	for _, item := range health {
		fmt.Println("Checks: ", item.Checks, item.Checks.AggregatedStatus())
		fmt.Println("Service: ", item.Service.ID, item.Service.Address, item.Service.Port)
		fmt.Println("--- ")
	}

	////////////////////

	// fire the gRPC server in a goroutine anonymous function
	go func() {
		// create a listener on TCP port
		lis, nErr := net.Listen("tcp", grpcAddr)
		if nErr != nil {
			log.Fatalf("Failed to listen gRPC: %v, port: %v", err, grpcAddr)
		}

		// create a gRPC server object
		// grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(AuthInterceptor)))
		grpcServer := grpc.NewServer(grpc.UnaryInterceptor(AuthInterceptor))
		// grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor))

		// attach the Ping service to the server
		pb.RegisterShippingServiceServer(grpcServer, &service{session, vesselClient})

		// Initialize all metrics.
		grpc_prometheus.Register(grpcServer)

		// start the server
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if gErr := grpcServer.Serve(lis); gErr != nil {
			log.Fatalf("Sailed to serve: %s", gErr)
		}
	}()

	// fire the REST server in a goroutine anonymous function
	go func() {
		// restAddress := fmt.Sprintf("%s:%d", "localhost", 8082)
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()
		// Setup the client gRPC options
		opts := []grpc.DialOption{grpc.WithInsecure()}

		// Register ping
		err = pb.RegisterShippingServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
		if err != nil {
			log.Fatalf("Could not register ShippingService: %s", err)
		}

		httpMux := http.NewServeMux()
		httpMux.Handle("/", mux)
		httpMux.Handle("/metrics", promhttp.Handler())
		httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if err := session.Ping(); err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		httpMux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
			dir := "./proto/consignment"
			if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
				log.Printf("Swagger Not Found: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			log.Printf("Serving Swagger %s", r.URL.Path)
			p := strings.TrimPrefix(r.URL.Path, "/swagger/")
			p = path.Join(dir, p)
			http.ServeFile(w, r, p)
		})

		s := &http.Server{
			Addr:    grpcHTTPAddr,
			Handler: allowCORS(httpMux),
		}

		log.Printf("Starting REST server on %s", grpcHTTPAddr)
		// http.ListenAndServe(restAddress, mux)
		s.ListenAndServe()
	}()

	// infinite loop
	log.Printf("Entering infinite loop")
	select {}

}
