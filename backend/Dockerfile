# Go backend — çok aşamalı derleme.
# imaging saf Go (cgo yok), o yüzden statik binary + küçük imaj mümkün.

# ---- derleme aşaması ----
FROM golang:1.25-alpine AS build

WORKDIR /src

# Bağımlılıkları önce çek — kaynak değişince katman cache'i korunsun
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO kapalı → tam statik binary, scratch/alpine'de çalışır
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# seed binary — prod'da ilk admin kullanıcısını oluşturmak için (etkileşimsiz mod)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/seed ./cmd/seed

# migrate CLI'yı da derle (go.mod'da yok, ayrı modül olarak alınıyor)
RUN CGO_ENABLED=0 go install -tags 'postgres' \
    github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1 \
    && cp "$(go env GOPATH)/bin/migrate" /out/migrate

# ---- çalışma aşaması ----
FROM alpine:3.20

# HTTPS çağrıları (R2 vb.) ve zaman dilimi için sertifikalar/tzdata
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app

WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /out/seed /app/seed
COPY --from=build /out/migrate /usr/local/bin/migrate
COPY migrations /app/migrations

# Görseller bu dizine yazılıyor — compose'ta volume olarak bağlanacak
RUN mkdir -p /app/uploads && chown -R app:app /app

USER app

EXPOSE 8080

CMD ["/app/server"]
