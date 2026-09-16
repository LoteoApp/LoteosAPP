# syntax=docker/dockerfile:1
#
# Production image for the frontend: builds the static bundle, then serves
# it from an unprivileged nginx (no root, no privileged port).
#
# Dokploy: Dockerfile Path = apps/frontend/Dockerfile, Docker Context Path = .
# VITE_* values are compile-time — Vite bakes them into the JS bundle, so
# they must be set as Build Time Arguments in Dokploy, not runtime
# environment variables. They end up in a public bundle either way, so they
# don't belong in Build-time Secrets.

FROM node:24-alpine AS builder
WORKDIR /app

RUN corepack enable && corepack prepare pnpm@10.30.1 --activate

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/frontend/package.json apps/frontend/package.json
RUN pnpm install --filter @loteos/frontend --frozen-lockfile

COPY . .

ARG VITE_API_URL
ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_ANON_KEY
ENV VITE_API_URL=${VITE_API_URL} \
    VITE_SUPABASE_URL=${VITE_SUPABASE_URL} \
    VITE_SUPABASE_ANON_KEY=${VITE_SUPABASE_ANON_KEY}

RUN pnpm --filter @loteos/frontend build

FROM nginxinc/nginx-unprivileged:1.29-alpine AS server
COPY apps/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/apps/frontend/dist /usr/share/nginx/html
EXPOSE 8080
