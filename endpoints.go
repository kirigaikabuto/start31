package start31

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	redis_lib "github.com/kirigaikabuto/common-lib31"
	users "github.com/kirigaikabuto/users31"
	"io/ioutil"
	"net/http"
	"time"
)

type HttpEndpoints interface {
	TestEndpoint() func(w http.ResponseWriter, r *http.Request)
	TestEndpointWithParam(idParam string) func(w http.ResponseWriter, r *http.Request)
	TestPostEndpoint() func(w http.ResponseWriter, r *http.Request)
	RegisterEndpoint() func(w http.ResponseWriter, r *http.Request)
	LoginEndpoint() func(w http.ResponseWriter, r *http.Request)
	ProfileEndpoint() func(w http.ResponseWriter, r *http.Request)
}

type httpEndpoints struct {
	//variable connection to db
	usersStore users.UsersStore
	redisStore *redis_lib.RedisStore
}

func NewHttpEndpoints(uS users.UsersStore, rS *redis_lib.RedisStore) HttpEndpoints {
	return &httpEndpoints{usersStore: uS, redisStore: rS}
}

func (h *httpEndpoints) TestEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := users.User{
			Id:        "1232213",
			Username:  "username123",
			Password:  "asdssadsa",
			FirstName: "1233",
			LastName:  "13213",
			Avatar:    "asdsadsadsadad",
		}
		respondJSON(w, http.StatusOK, user)
		return
	}
}

func (h *httpEndpoints) TestEndpointWithParam(idParam string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr, ok := vars[idParam]
		if !ok {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    "Dont have user with that id",
				StatusCode: http.StatusBadRequest,
			})
		}
		var response users.User
		usersData := []users.User{
			{
				Id:        "1",
				Username:  "1",
				Password:  "1",
				FirstName: "1",
				LastName:  "1",
				Avatar:    "1",
			},
			{
				Id:        "2",
				Username:  "2",
				Password:  "2",
				FirstName: "2",
				LastName:  "2",
				Avatar:    "2",
			},
		}
		if idStr == "1" {
			response = usersData[0]
		} else if idStr == "2" {
			response = usersData[1]
		} else {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    "Dont have user with that id",
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		respondJSON(w, http.StatusOK, response)
		return
	}
}

func (h *httpEndpoints) TestPostEndpoint() func(w http.ResponseWriter, r *http.Request) {
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
		user.Id = "3333"
		respondJSON(w, http.StatusCreated, user)
		return
	}
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
		err = h.redisStore.SetValue(key, user, 5*time.Minute)
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
