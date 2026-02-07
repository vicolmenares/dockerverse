# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Build argument for API URL
ARG PUBLIC_API_URL=http://localhost:3001
ENV PUBLIC_API_URL=$PUBLIC_API_URL

# Copy package files (ignoring lockfile)
COPY package.json ./

# Install dependencies fresh
RUN npm install --legacy-peer-deps

# Copy source
COPY . .

# Build the app with the API URL
RUN echo "Building with PUBLIC_API_URL=$PUBLIC_API_URL" && npm run build

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 sveltekit

# Copy built app
COPY --from=builder --chown=sveltekit:nodejs /app/build ./build
COPY --from=builder --chown=sveltekit:nodejs /app/package*.json ./
COPY --from=builder --chown=sveltekit:nodejs /app/node_modules ./node_modules

USER sveltekit

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

CMD ["node", "build"]
