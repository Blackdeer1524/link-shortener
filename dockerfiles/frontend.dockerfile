FROM node:22-alpine3.19

EXPOSE 8000
COPY package.json package-lock.json ./

RUN npm install 

COPY . ./
RUN npm install 
ENTRYPOINT ["npm", "run", "dev"]
