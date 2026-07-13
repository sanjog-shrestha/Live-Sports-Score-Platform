# ---- Build Stage ----
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./ 
RUN go build -o server . 

# ---- Run stage ----
FROM alpine:3.19 
WORKDIR /app 
COPY --from=build /app/server . 
COPY data.json .
EXPOSE 8080
CMD ["./server"]