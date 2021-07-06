package start31

import (
	"encoding/json"
	"github.com/djumanoff/amqp"
	"github.com/google/uuid"
	redis_lib "github.com/kirigaikabuto/common-lib31"
	users "github.com/kirigaikabuto/users31"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	createUserAmqpEndpoint   = "users.create"
	getByUsernameAndPassword = "users.getByUsernameAndPassword"
	getById                  = "users.getById"
)

type HttpEndpoints interface {
	RegisterEndpoint() func(w http.ResponseWriter, r *http.Request)
	LoginEndpoint() func(w http.ResponseWriter, r *http.Request)
	ProfileEndpoint() func(w http.ResponseWriter, r *http.Request)
}

type httpEndpoints struct {
	//variable connection to db
	amqpConnect amqp.Client
	redisStore  *redis_lib.RedisConnectStore
}

func NewHttpEndpoints(cl amqp.Client, rS *redis_lib.RedisConnectStore) HttpEndpoints {
	return &httpEndpoints{amqpConnect: cl, redisStore: rS}
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
		data, err := h.amqpConnect.Call(createUserAmqpEndpoint, amqp.Message{Body: jsonData})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		createdUser := &users.User{}
		err = json.Unmarshal(data.Body, createdUser)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusCreated, createdUser)
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
		data, err := h.amqpConnect.Call(getByUsernameAndPassword, amqp.Message{Body: jsonData})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		user := &users.User{}
		err = json.Unmarshal(data.Body, user)
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
		cmd := &users.User{Id: userId}
		dataJson, err := json.Marshal(cmd)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		data, err := h.amqpConnect.Call(getById, amqp.Message{Body: dataJson})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		user := &users.User{}
		err = json.Unmarshal(data.Body, &user)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusOK, user)
		return
	}
}
