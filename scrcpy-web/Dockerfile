FROM node:24-trixie-slim AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 make g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:24-trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    android-tools-adb ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev --omit=optional && npm cache clean --force
COPY --from=build /app/dist ./dist
COPY --from=build /app/assets ./assets
EXPOSE 8000
CMD ["node", "dist/index.js"]
