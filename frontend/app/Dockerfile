# Nuxt public site — SSR (spec §5.1). Statik değil, sürekli çalışan node sunucusu.
# Çok aşamalı: bağımlılık → build → yalın çalışma imajı.

# ---- bağımlılıklar ----
FROM node:22-alpine AS deps

# @nuxt/image sharp kullanıyor; native binary'ler için build araçları
RUN apk add --no-cache libc6-compat
RUN corepack enable

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
# Build için tüm bağımlılıklar (dev dahil) gerekli
RUN pnpm install --frozen-lockfile

# ---- build ----
FROM node:22-alpine AS build

RUN corepack enable
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Nuxt .output/ üretir — Nitro sunucusu + istemci varlıkları
# NODE_OPTIONS: küçük VPS'lerde (ör. 2GB RAM) V8'in varsayılan heap limiti
# build'i "JavaScript heap out of memory" ile düşürüyor — swap varsa bile.
ENV NODE_OPTIONS="--max-old-space-size=2048"
RUN pnpm build

# ---- çalışma ----
FROM node:22-alpine AS runtime

RUN adduser -D -u 10001 app
WORKDIR /app

# Nitro .output tek başına yeterli — node_modules gerekmiyor
COPY --from=build /app/.output ./.output

USER app

ENV PORT=3000
ENV HOST=0.0.0.0
EXPOSE 3000

CMD ["node", ".output/server/index.mjs"]
