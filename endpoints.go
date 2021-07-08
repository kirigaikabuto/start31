package start31

import (
	"encoding/json"
	"github.com/djumanoff/amqp"
	"github.com/google/uuid"
	redis_lib "github.com/kirigaikabuto/common-lib31"
	"github.com/kirigaikabuto/orders31"
	"github.com/kirigaikabuto/products31"
	users "github.com/kirigaikabuto/users31"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	createUserAmqpEndpoint    = "users.create"
	getByUsernameAndPassword  = "users.getByUsernameAndPassword"
	getById                   = "users.getById"
	createProductAmqpEndpoint = "products.create"
	listProductAmqpEndpoint   = "products.list"
	createOrderAmqpEndpoint   = "orders.create"
	listOrderAmqpEndpoint     = "orders.list"
)

type HttpEndpoints interface {
	RegisterEndpoint() func(w http.ResponseWriter, r *http.Request)
	LoginEndpoint() func(w http.ResponseWriter, r *http.Request)
	ProfileEndpoint() func(w http.ResponseWriter, r *http.Request)
	CreateProductEndpoint() func(w http.ResponseWriter, r *http.Request)
	ListProductEndpoint() func(w http.ResponseWriter, r *http.Request)
	CreateOrder() func(w http.ResponseWriter, r *http.Request)
	ListOrder() func(w http.ResponseWriter, r *http.Request)
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

func (h *httpEndpoints) CreateProductEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		product := &products31.Product{}
		err = json.Unmarshal(jsonData, &product)
		if product.Name == "" {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    "Please write name",
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		data, err := h.amqpConnect.Call(createProductAmqpEndpoint, amqp.Message{Body: jsonData})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		createdProduct := &products31.Product{}
		err = json.Unmarshal(data.Body, &createdProduct)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusCreated, createdProduct)
		return
	}
}

func (h *httpEndpoints) ListProductEndpoint() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := h.amqpConnect.Call(listProductAmqpEndpoint, amqp.Message{})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		products := []products31.Product{}
		err = json.Unmarshal(data.Body, &products)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusOK, products)
		return
	}
}

func (h *httpEndpoints) CreateOrder() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		contextData := r.Context().Value("user_id")
		userId := contextData.(string)
		jsonData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}
		cmd := &orders31.Order{}
		err = json.Unmarshal(jsonData, &cmd)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		cmd.UserId = userId
		req, err := json.Marshal(cmd)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		resJson, err := h.amqpConnect.Call(createOrderAmqpEndpoint, amqp.Message{Body: req})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		newOrder := &orders31.Order{}
		err = json.Unmarshal(resJson.Body, &newOrder)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusCreated, newOrder)
		return
	}
}

func (h *httpEndpoints) ListOrder() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		contextData := r.Context().Value("user_id")
		userId := contextData.(string)
		cmd := &orders31.Order{UserId: userId}
		req, err := json.Marshal(cmd)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		resJson, err := h.amqpConnect.Call(listOrderAmqpEndpoint, amqp.Message{Body: req})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		orders := &[]orders31.Order{}
		err = json.Unmarshal(resJson.Body,&orders)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, HttpError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}
		respondJSON(w, http.StatusCreated, orders)
		return
	}
}
