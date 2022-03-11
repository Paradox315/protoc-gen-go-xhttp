# protoc-gen-go-xhttp
## 1 Introduction
> protoc-gen-go-xhttp is a protoc plugin that generates Go code for web services.
## 2 Installation
> go install github.com/Paradox315/protoc-gen-go-xhttp@latest
## 3 Usage
### 3.1 Generating Go code
```makefile
protoc  --proto_path=. --go-xhttp_out=paths=source_relative:. $(API_PROTO_FILES)
```
before running the above command, you need to install the protoc-gen-go-xhttp plugin,and config the http rule in your protobuf file.

Just like this:
```protobuf
  rpc List (PageRequest) returns (HelloReply)  {
    option (google.api.http) = {
      get: "/list"
    };
}
```
## 3.2 Generating Go code with annotations

This plugin supports the following annotations,you can use them to generate your code just like spring boot:
1. `@Path("<router-prefix>")`
This annotation is used to set the router prefix,the default value is `/`.
2. `@Operations`
This annotation is used to set the router operations,mostly used for user data collection.
3. `@Validate`
This annotation is used to set the validation rules.
4. `@Custom("<custom-annotation-name>")`
This annotation is used to set the custom annotation.

The example below is the protobuf file:
```protobuf
// The greeting service definition.
// @Path("demo")
service Greeter {
  // Gets a greeting for a user.
  // @Validate
  rpc List (PageRequest) returns (HelloReply)  {
    option (google.api.http) = {
      get: "/list"
    };
  }

  // Gets a greeting for a user.
  rpc Get (HelloRequest) returns (HelloReply)  {
    option (google.api.http) = {
      get: "/get/{name}"
    };
  }

  // Adds a greeting for a user.
  // @Validate
  rpc Add (HelloRequest) returns (HelloReply)  {
    option (google.api.http) = {
      post: "/",
      body: "*"
    };
  }
}
```