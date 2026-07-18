# ---- Build Stage ----
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY *.go ./ 
RUN go mod tidy 
RUN go build -o server . 

# ---- Run stage ----
FROM alpine:3.19 
WORKDIR /app 
COPY --from=build /app/server . 
COPY data.json .
COPY static ./static
EXPOSE 8080
CMD ["./server"]