FROM maven:3.8.6-openjdk-11 AS builder
WORKDIR /app
ARG mapperName=mapper
COPY pom.xml .
COPY src ./src
RUN mvn clean package -D mapperName=${mapperName}

FROM openjdk:11
WORKDIR /app
ARG mapperName=mapper
COPY --from=builder /app/src/ /app/src
COPY --from=builder /app/target/${mapperName}.jar /app/target/${mapperName}.jar

ENV MAPPER_NAME=${mapperName}

CMD sh -c "java -jar /app/target/${MAPPER_NAME}.jar"