package start31

import (
	"encoding/json"
	"github.com/google/uuid"
	redis_lib "github.com/kirigaikabuto/common-lib31"
	users "github.com/kirigaikabuto/users31"
	"io/ioutil"
	"net/http"
	"time"
)

type HttpEndpoints interface {
	RegisterEndpoint() func(w http.ResponseWriter, r *http.Request)
	LoginEndpoint() func(w http.ResponseWriter, r *http.Request)
	ProfileEndpoint() func(w http.ResponseWriter, r *http.Request)
}

type httpEndpoints struct {
	//variable connection to db
	usersStore users.UsersStore
	redisStore *redis_lib.RedisConnectStore
}

func NewHttpEndpoints(uS users.UsersStore, rS *redis_lib.RedisConnectStore) HttpEndpoints {
	return &httpEndpoints{usersStore: uS, redisStore: rS}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func (h *httpEndpoints) RegisterEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		user := &users.User{}
		err = json.Unmarshal(jsonData, &user)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		if user.Username == "" || user.Password == "" {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    ErrUsernamePasswordEmpty.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		oldUser, err := h.usersStore.GetByUsernameAndPassword(user.Username, user.Password)
		if err != nil && err != users.ErrNoUser {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		if oldUser != nil {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    ErrUserAlreadyExist.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		response, err := h.usersStore.Create(user)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusCreated, response)
		return
	}
}

func (h *httpEndpoints) LoginEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		req := &LoginRequest{}
		err = json.Unmarshal(jsonData, &req)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		user, err := h.usersStore.GetByUsernameAndPassword(req.Username, req.Password)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		key := uuid.New().String()
		err = h.redisStore.Save(key, user.Id, 5*time.Minute)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		response := &LoginResponse{AccessKey: key}
		respondJSON(w, http.StatusOK, response)
		return
	}
}

func (h *httpEndpoints) ProfileEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		contextData := r.Context().Value("user_id")
		userId := contextData.(string)
		response, err := h.usersStore.Get(userId)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusOK, response)
		return
	}
}
