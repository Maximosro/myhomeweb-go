# ============================================
# Stage 1: Build — compila el binario Go
# ============================================
FROM golang:1.26-alpine AS build

WORKDIR /app

# Copiar dependencias primero para cachear la capa
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar binario estático (CGO_ENABLED=0 porque modernc.org/sqlite es pure-Go)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o myhomeweb .

# ============================================
# Stage 2: Runtime — Alpine mínimo con healthcheck
# ============================================
FROM alpine:3.21 AS runtime

WORKDIR /app

# Instalar wget (healthcheck) + ca-certificates (JWT validation vía HTTPS)
RUN apk add --no-cache wget ca-certificates

# Crear usuario no-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copiar binario compilado desde stage 1
COPY --from=build /app/myhomeweb .

# Copiar assets necesarios en runtime
COPY --from=build /app/static/ static/
COPY --from=build /app/templates/ templates/
COPY --from=build /app/data.sql .

# Crear directorio de datos y asignar propiedad
RUN mkdir -p /data && chown -R appuser:appgroup /app /data

# Cambiar a usuario no-root
USER appuser:appgroup

# Exponer puerto de la app (solo documentación)
EXPOSE 19484

# Configurar path de BD dentro del volumen
ENV DB_PATH=/data/myhomeweb.db

# Healthcheck: verifica que la raíz responde (no hay endpoint /health dedicado)
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:19484/health || exit 1

# Punto de entrada
ENTRYPOINT ["./myhomeweb"]
