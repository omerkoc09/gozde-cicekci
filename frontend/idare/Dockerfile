# Admin panel — Vite SPA (statik). Prod'da /idare alt yolunda servis edilir.
# Bu imaj sadece build üretir; statik çıktı bir volume'a kopyalanıp Caddy servis eder.

# ---- build ----
FROM node:22-alpine AS build

RUN apk add --no-cache libc6-compat
RUN corepack enable

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
# --ignore-scripts: postinstall build:icons kaynak (src/) ister ama henüz
# kopyalanmadı. İkonlar prebuild'de zaten üretiliyor.
RUN pnpm install --frozen-lockfile --ignore-scripts

COPY . .

# base=/idare/ ile derle: router ve varlık yolları alt yola göre üretilir.
# prebuild build:icons ve msw:init'i çalıştırır (kaynak artık mevcut).
ENV VITE_BASE=/idare/
RUN pnpm build

# ---- çıktı taşıyıcı ----
# Çalışan bir sunucu değil; compose bu imajdan dist/'i shared volume'a kopyalar.
FROM alpine:3.20 AS dist

WORKDIR /dist
COPY --from=build /app/dist /dist

# Compose entrypoint'i dist/'i /output volume'una kopyalar
CMD ["sh", "-c", "cp -r /dist/. /output/ && echo 'admin panel dist kopyalandı'"]
