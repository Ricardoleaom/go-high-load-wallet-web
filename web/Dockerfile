# Build the static console.
FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Serve it from nginx, unprivileged (no root, binds 8080) to match the
# backend's own distroless, non-root containers. Vite's defaults already
# point at the backend, Keycloak and the load lab on their Compose ports,
# so no build args are needed for the stack in ../compose.yaml.
FROM nginxinc/nginx-unprivileged:1.30-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
