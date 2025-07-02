package server

import (
	"fmt"
	"net"
	"net/http"
)

func ListenAndServe() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("Server listening on port %d\n", port)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/oauth/github/callback", getGithubCallback)
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	if err := http.Serve(listener, nil); err != nil {
		panic(err)
	}
	return nil
}

func getGithubCallback(w http.ResponseWriter, r *http.Request) {
	// q := r.URL.Query()
	// state := q.Get("state")
	// code := q.Get("code")

	// if state == "" || code == "" {
	// 	// todo: handle here
	// 	// return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Invalid authorization. You are malicious")
	// }

	// token, err := models.OAuthConf.Exchange(c.Context(), code)
	// if err != nil {
	// 	SessionIdCache.Replace(state, "failed", 30*time.Second)
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	// }

	// client := models.OAuthConf.Client(context.Background(), token)
	// resp, err := client.Get("https://api.github.com/user")
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	SessionIdCache.Replace(state, "failed", 30*time.Second)
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to obtain your user info")
	// }
	// defer resp.Body.Close()

	// var user struct {
	// 	Login string `json:"login"`
	// 	Email string `json:"email"`
	// }

	// if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
	// 	slog.Error(err.Error())
	// 	SessionIdCache.Replace(state, "failed", 30*time.Second)
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to parse user info")
	// }
	// url := "https://api.github.com/repos/nikumar1206/loco/collaborators/" + user.Login
	// resp, err = client.Get(url)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	SessionIdCache.Replace(state, "failed", 30*time.Second)
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to obtain your user info")
	// }

	// defer resp.Body.Close()

	// if resp.StatusCode != 204 {
	// 	slog.Error("unexpected sc from github api", slog.Int("sc", resp.StatusCode))
	// 	SessionIdCache.Replace(state, "failed", 30*time.Second)
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to parse collaborators response")
	// }

	// middlewares.TokenCache.Set(token.AccessToken, user.Login, time.Until(token.Expiry.Add(-10*time.Minute)))
	// err = SessionIdCache.Replace(state, token.AccessToken, 30*time.Second)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to find state in cache.")
	// }
	// return fmt.Printf("validated successfully")
	// w.WriteHeader(200)
	// w.Write([]byte("hello"))
}
