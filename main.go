package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type TelemetryPayload struct {
	DeviceID  string  `json:"device_id"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type Server struct {
	sqsClient *sqs.Client
	queueURL  string
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) telemetryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var p TelemetryPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if p.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	bodyBytes, _ := json.Marshal(p)

	if s.sqsClient != nil && s.queueURL != "" {
		ctx := context.TODO()
		_, err := s.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(s.queueURL),
			MessageBody: aws.String(string(bodyBytes)),
		})
		if err != nil {
			log.Printf("failed to send message to SQS: %v", err)
			http.Error(w, "failed to queue telemetry", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "device_id": p.DeviceID})
}

func main() {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	var sqsClient *sqs.Client
	if err == nil {
		sqsClient = sqs.NewFromConfig(cfg)
	} else {
		log.Printf("AWS config init notice: %v", err)
	}

	queueURL := os.Getenv("SQS_QUEUE_URL")

	srv := &Server{
		sqsClient: sqsClient,
		queueURL:  queueURL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.healthHandler)
	mux.HandleFunc("/telemetry", srv.telemetryHandler)

	log.Println("Starting telemetry-api on :8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}