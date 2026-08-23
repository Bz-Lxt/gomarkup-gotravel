FROM mcr.microsoft.com/playwright:v1.49.1-jammy
WORKDIR /tests
COPY package.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY . .
ENV E2E_BASE=http://frontend
CMD ["npx", "playwright", "test"]
