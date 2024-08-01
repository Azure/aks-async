# Shared Libraries

The toolkit is currently a collection of libraries that the rest of the mygreeter service can use.

In the future these libraries would be exposed and put on github or an external repository.

## Structure

### Database

A simple wrapper that will allow you to connect and query a database easier.

### OperationsBus

The interfaces that will allow you to create your own operations and run them on your own async. You will have to implement the receiver interface in order to run the operations, and each operation will have to implement the operation_handler interface in order to be run.

### Service Bus

A simple wrapper that will allow you to connect and query a service bus easier.
