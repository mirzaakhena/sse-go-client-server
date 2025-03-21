# Start from the official Golang image
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /nms

# Copy go.work and all go.mod files
COPY go.work ./
COPY nms/go.mod nms/
# COPY bigboard/go.mod bigboard/
# COPY dashboard/go.mod dashboard/
# COPY iam/go.mod iam/

# Download dependencies
RUN go work use -r ./
RUN go mod download

# Copy the source code
COPY . .

# Build all applications
RUN go build -o /nms/bin/app ./nms
# RUN go build -o /app/bin/bigboard ./bigboard
# RUN go build -o /app/bin/dashboard ./dashboard
# RUN go build -o /app/bin/iam ./iam

# Start a new stage from scratch
FROM alpine:latest

# Copy the built executables from the previous stage
COPY --from=builder /nms/bin/nms /nms/
# COPY --from=builder /app/bin/bigboard /app/
# COPY --from=builder /app/bin/dashboard /app/
# COPY --from=builder /app/bin/iam /app/

# Set the working directory
WORKDIR /nms

# Create an entrypoint script
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'case "$1" in' >> /entrypoint.sh && \
    echo '  "nms") exec ./nms ;;' >> /entrypoint.sh && \
    # echo '  "bigboard") exec ./bigboard ;;' >> /entrypoint.sh && \
    # echo '  "dashboard") exec ./dashboard ;;' >> /entrypoint.sh && \
    # echo '  "iam") exec ./iam ;;' >> /entrypoint.sh && \
    echo '  *) echo "Invalid application name. Use nms." && exit 1 ;;' >> /entrypoint.sh && \
    echo 'esac' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Default command (can be overridden)
CMD ["nms"]