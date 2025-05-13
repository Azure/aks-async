# Shared Libraries

The toolkit is currently a collection of libraries that the rest of the mygreeter service can use.

In the future these libraries would be exposed and put on github or an external repository.

## Structure

### Database

A simple wrapper that will allow you to connect and query a database easier.

Sample usage:
```go
dbClient, err = database.NewDbClient(context.Background(), databaseServerUrl, databasePort, databaseName)
if err != nil {
    logger.Error("Error creating connection pool: " + err.Error())
}

// The query is parametrized for you.
query := "SELECT LastName FROM family WHERE FirstName = ?"
rows, err := database.QueryDb(ctx, dbClient, query, firstName)
if err != nil {
    fmt.Println("Error checking if the previous operation of the entity is finished: " + err.Error())
}

var lastName string
for rows.Next() {
    err = rows.Scan(&lastName)
    if err != nil {
        fmt.Println("Error getting the lastName of the family: " + err.Error())
    }
}

fmt.Println("The last name of the family is: " + lastName)
```

### OperationsBus

This package holds the interfaces and methods that will allow you to create your own asynchronous operations, and have an asynchronous processor that runs them as they are received. This package assumes the existance of: a Service Bus to receive the messaages (currently only supports Azure Service Bus), a database where you store entity information, a database where you store operation information. All these requirements are implemented by the user by using the different interfaces that are provided.

Sample usage:
```go
ctx, cancel := context.WithCancel(context.Background())

// Instantiate a matcher. Here we would store all of our operation types.
matcher := operationsbus.NewMatcher()
lro := &LongRunningOperation{}
sro := &ShortRunningOperation{}
matcher.Register(lro.GetName(ctx), lro)
matcher.Register(sro.GetName(ctx), sro)

processor, err := processor.CreateProcessor(receiver, matcher, operationContainerClient, entityController, logger, handler, nil, hooks)

// Start processing the operations.
err = asyncStruct.Processor.Start(ctx)
if err != nil {
    cancel()
}
cancel()
```

In order to create a new operation type, you will simply need to create a struct that is of implements the interface `ApiOperation` and another struct representing the modified entity that implementes the `Entity` interface.

Here's a quick example: 
```go
// Example implementation of ApiOperation
var _ operation.ApiOperation = &SampleOperation{}

type SampleOperation struct {
	opReq operation.OperationRequest
	Num   int
}

func (l *SampleOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, error) {
	if opReq.OperationId == "1" {
		return nil, errors.New("No OperationId")
	}
	l.opReq = opReq
	l.Num = 1
	return &l, nil
}

func (l *SampleOperation) GuardConcurrency(ctx context.Context, entityInstance entity.Entity) *entity.CategorizedError {
	if l.opReq.OperationId == "2" {
		ce := &entity.CategorizedError{Err: errors.New("Incorrect OperationId")}
		return ce
	}
	return nil
}

func (l *SampleOperation) Run(ctx context.Context) error {
	if l.opReq.OperationId == "3" {
		return errors.New("Incorrect OperationId")
	}
	return nil
}

func (l *SampleOperation) GetOperationRequest() *operation.OperationRequest {
	return &l.opReq
}

// Example implementatin of Entity
type TestEntity struct {
	latestOperationId string
}

func (e *TestEntity) GetLatestOperationID() string {
	return e.latestOperationId
}

func NewTestEntity(latestOperationId string) *TestEntity {
	return &TestEntity{
		latestOperationId: latestOperationId,
	}
}
```

Additionally, if there are fields that you need to Init your operation, but they don't currently exist in the OperationRequest, you can use the `Extension` variable to add any interface you need together with the `SetExtension(interface{})` method in order to use that interface as a more concrete type and directly change the OperationRequest.Extension variable, so you can continue using the same instance.
```go
type Sample struct {
	Message string
	Num     int
}

var body OperationRequest
err := json.Unmarshal(marshalledOperation, &body)
if err != nil {
    t.Fatalf("Could not unmarshall operation request:" + err.Error())
}

// SetExtension(interface{}) uses the type of a parameter that is passed in to instantiate the Extension into the correct type you need.
s := &Sample{}
err = body.SetExtension(s)
if err != nil {
    t.Fatalf("SetExtension errored: " + err.Error())
}

// Check if the type and value are correctly set
if ext, ok := body.Extension.(*Sample); ok {
    fmt.Println(ext.Message)
    fmt.Println(ext.Num)
} else {
    fmt.Println("Extension is not of type *Sample")
}
```

### Service Bus

A simple wrapper that will allow you to connect and receive messages from a service bus client.

Sample usage:
```go
ctx := context.Background()
sender, err := serviceBusClient.NewServiceBusSender(ctx, queueName)
if err != nil {
    fmt.Println("Something went wrong creating the service bus sender: " + err.Error())
}

expirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
extension := "Hello!"
operation := operationsbus.NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, expirationTime, nil, "", extension) 

marshalledOperation, err := json.Marshal(operation)
if err != nil {
    fmt.Println("Error marshalling operation: " + err.Error())
}

err = sender.SendMessage(ctx, marshalledOperation)
if err != nil {
    fmt.Println("Something happened: " + err.Error())
}
```

## Util

### Mock

Run the following to create mocks of an interface, with a sample of the service bus interface:
```bash
mockgen -source=servicebus/servicebus_interface.go -destination=mocks/mock_service_bus.go -package=mocks
```
