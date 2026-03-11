FROM node:22-alpine AS deps
WORKDIR /repo
RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/admin-web/package.json apps/admin-web/package.json
COPY packages/sdk-ts/package.json packages/sdk-ts/package.json
COPY packages/shared-types/package.json packages/shared-types/package.json

RUN pnpm install --filter @opener-netdoor/admin-web --filter @opener-netdoor/sdk-ts --filter @opener-netdoor/shared-types --frozen-lockfile

FROM node:22-alpine AS build
WORKDIR /repo
RUN corepack enable
ARG NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1
ARG NEXT_PUBLIC_DEFAULT_SCOPE_ID=default
ARG NEXT_PUBLIC_SINGLE_OWNER_MODE=true
ENV NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
ENV NEXT_PUBLIC_DEFAULT_SCOPE_ID=${NEXT_PUBLIC_DEFAULT_SCOPE_ID}
ENV NEXT_PUBLIC_SINGLE_OWNER_MODE=${NEXT_PUBLIC_SINGLE_OWNER_MODE}

COPY --from=deps /repo/node_modules ./node_modules
COPY --from=deps /repo/apps/admin-web/node_modules ./apps/admin-web/node_modules
COPY --from=deps /repo/packages/sdk-ts/node_modules ./packages/sdk-ts/node_modules
COPY --from=deps /repo/packages/shared-types/node_modules ./packages/shared-types/node_modules
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/admin-web ./apps/admin-web
COPY packages/sdk-ts ./packages/sdk-ts
COPY packages/shared-types ./packages/shared-types

RUN pnpm --filter @opener-netdoor/admin-web build

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

COPY --from=build /repo/apps/admin-web/.next/standalone ./
COPY --from=build /repo/apps/admin-web/.next/static ./apps/admin-web/.next/static
COPY --from=build /repo/apps/admin-web/public ./apps/admin-web/public

EXPOSE 3000
CMD ["node", "apps/admin-web/server.js"]
