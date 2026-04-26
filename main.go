package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type config struct {
	GreenAPIHost string
}

type credentials struct {
	IDInstance       string `json:"idInstance"`
	APITokenInstance string `json:"apiTokenInstance"`
}

type sendMessageRequest struct {
	credentials
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

type sendFileRequest struct {
	credentials
	ChatID   string `json:"chatId"`
	URLFile  string `json:"urlFile"`
	FileName string `json:"fileName"`
}

var appConfig = loadConfig()

func main() {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/api/getSettings", withMethod(http.MethodPost, handleGetSettings))
	mux.HandleFunc("/api/getStateInstance", withMethod(http.MethodPost, handleGetStateInstance))
	mux.HandleFunc("/api/sendMessage", withMethod(http.MethodPost, handleSendMessage))
	mux.HandleFunc("/api/sendFileByUrl", withMethod(http.MethodPost, handleSendFileByURL))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Server started at http://localhost:%s", port)
	if appConfig.GreenAPIHost != "" {
		log.Printf("GREEN-API host override: %s", appConfig.GreenAPIHost)
	} else {
		log.Printf("GREEN-API host mode: derive from idInstance prefix")
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func loadConfig() config {
	host := strings.TrimSpace(os.Getenv("GREEN_API_HOST"))
	return config{
		GreenAPIHost: host,
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("incoming request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	respBody, statusCode, err := callGreenAPI(req, "getSettings", nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSONRaw(w, statusCode, respBody)
}

func handleGetStateInstance(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	respBody, statusCode, err := callGreenAPI(req, "getStateInstance", nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSONRaw(w, statusCode, respBody)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.ChatID) == "" || strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "chatId and message are required")
		return
	}

	payload := map[string]string{
		"chatId":  req.ChatID,
		"message": req.Message,
	}

	respBody, statusCode, err := callGreenAPI(req.credentials, "sendMessage", payload)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSONRaw(w, statusCode, respBody)
}

func handleSendFileByURL(w http.ResponseWriter, r *http.Request) {
	var req sendFileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.ChatID) == "" || strings.TrimSpace(req.URLFile) == "" {
		writeError(w, http.StatusBadRequest, "chatId and urlFile are required")
		return
	}
	if strings.TrimSpace(req.FileName) == "" {
		req.FileName = "file"
	}

	payload := map[string]string{
		"chatId":   req.ChatID,
		"urlFile":  req.URLFile,
		"fileName": req.FileName,
	}

	respBody, statusCode, err := callGreenAPI(req.credentials, "sendFileByUrl", payload)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSONRaw(w, statusCode, respBody)
}

func withMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next(w, r)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	switch v := dst.(type) {
	case *credentials:
		if strings.TrimSpace(v.IDInstance) == "" || strings.TrimSpace(v.APITokenInstance) == "" {
			return fmt.Errorf("idInstance and apiTokenInstance are required")
		}
	case *sendMessageRequest:
		if strings.TrimSpace(v.IDInstance) == "" || strings.TrimSpace(v.APITokenInstance) == "" {
			return fmt.Errorf("idInstance and apiTokenInstance are required")
		}
	case *sendFileRequest:
		if strings.TrimSpace(v.IDInstance) == "" || strings.TrimSpace(v.APITokenInstance) == "" {
			return fmt.Errorf("idInstance and apiTokenInstance are required")
		}
	}

	return nil
}

func callGreenAPI(creds credentials, method string, payload any) ([]byte, int, error) {
	host := resolveGreenAPIHost(creds.IDInstance)
	url := fmt.Sprintf(
		"https://%s/waInstance%s/%s/%s",
		host,
		creds.IDInstance,
		method,
		creds.APITokenInstance,
	)
	log.Printf("green-api request: method=%s url=%s", requestMethodFromPayload(payload), url)
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	requestMethod := http.MethodGet
	if payload != nil {
		requestMethod = http.MethodPost
	}

	req, err := http.NewRequest(requestMethod, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("green-api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read green-api response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func resolveGreenAPIHost(idInstance string) string {
	if appConfig.GreenAPIHost != "" {
		return appConfig.GreenAPIHost
	}

	trimmed := strings.TrimSpace(idInstance)
	if len(trimmed) >= 4 {
		return fmt.Sprintf("%s.api.green-api.com", trimmed[:4])
	}

	return "api.green-api.com"
}

func requestMethodFromPayload(payload any) string {
	if payload != nil {
		return http.MethodPost
	}
	return http.MethodGet
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONRaw(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}
