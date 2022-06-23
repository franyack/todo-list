// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	"net/http"
	"sync"
	"sync/atomic"
	"todo-list/models"
	"todo-list/restapi/operations/todos"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"todo-list/restapi/operations"
)

//go:generate swagger generate server --target ../../todo-list --name TodoList --spec ../swagger.yml --principal interface{}

func configureFlags(api *operations.TodoListAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

var items = make(map[int64]*models.Item)
var lastID int64
var itemsLock = &sync.Mutex{}

func newItemID() int64 {
	return atomic.AddInt64(&lastID, 1)
}

func addItem(item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	newID := newItemID()
	item.ID = newID
	items[newID] = item

	return nil
}

func updateItem(id int64, item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	item.ID = id
	items[id] = item
	return nil
}

func deleteItem(id int64) error {
	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	delete(items, id)
	return nil
}

func allItems(since int64, limit int32) (result []*models.Item) {
	result = make([]*models.Item, 0)
	for id, item := range items {
		if len(result) >= int(limit) {
			return
		}
		if since == 0 || id > since {
			result = append(result, item)
		}
	}
	return result
}

func configureAPI(api *operations.TodoListAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	Configure(api)

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

func Configure(api *operations.TodoListAPI) {
	api.TodosAddOneHandler = todos.AddOneHandlerFunc(func(params todos.AddOneParams) middleware.Responder {
		if err := addItem(params.Body); err != nil {
			return todos.NewAddOneDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				Code:    http.StatusInternalServerError,
				Message: swag.String(err.Error()),
			})
		}
		return todos.NewAddOneCreated().WithPayload(params.Body)
	})

	api.TodosDestroyOneHandler = todos.DestroyOneHandlerFunc(func(params todos.DestroyOneParams) middleware.Responder {
		if err := deleteItem(params.ID); err != nil {
			return todos.NewDestroyOneDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				Code:    http.StatusInternalServerError,
				Message: swag.String(err.Error()),
			})
		}
		return todos.NewDestroyOneNoContent()
	})

	api.TodosFindTodosHandler = todos.FindTodosHandlerFunc(func(params todos.FindTodosParams) middleware.Responder {
		mergedParams := todos.NewFindTodosParams()
		mergedParams.Since = swag.Int64(0)
		if params.Since != nil {
			mergedParams.Since = params.Since
		}
		if params.Limit != nil {
			mergedParams.Limit = params.Limit
		}
		return todos.NewFindTodosOK().WithPayload(allItems(*mergedParams.Since, *mergedParams.Limit))
	})

	api.TodosUpdateOneHandler = todos.UpdateOneHandlerFunc(func(params todos.UpdateOneParams) middleware.Responder {
		if err := updateItem(params.ID, params.Body); err != nil {
			s := err.Error()
			return todos.NewUpdateOneDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				Code:    http.StatusInternalServerError,
				Message: &s,
			})
		}
		return todos.NewUpdateOneOK().WithPayload(params.Body)
	})
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
