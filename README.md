## **Token Management Server 🚀**
### Token Management System Architecture

#### Overview

The Token Management System (TMS) is designed to handle the lifecycle of tokens within a distributed environment. It offers APIs for generating, assigning, extending expiry, unblocking, and deleting tokens. The system uses Redis for high-speed token storage and management, taking advantage of Redis’ fast operations to support concurrent access and efficient expiry management. Additionally, an Expiry Manager runs in the background, ensuring the cleanup of expired tokens and maintaining system performance.
#### Core Features

- **Token Generation:** Generate a new token and store it in Redis.
- **Token Assignment:** Assign an available token to a client and lock it using Redis `SETNX`.
- **Token Expiry Management:** Extend the expiry of a token on client request to prevent premature expiration.
- **Token Unblocking:** Return a token to the available pool for future use.
- **Token Deletion:** Permanently remove a token from the system.

#### System Components

1. **Client Requests**  
   The following API endpoints allow clients to interact with the system:
   - **POST /generate-token:** Request to generate a new token.
   - **POST /assign-token:** Request to assign an available token to a client.
   - **POST /keep-alive:** Extend the expiry of an assigned token.
   - **POST /unblock-token:** Return a token to the available pool.
   - **DELETE /delete-token:** Delete a token permanently from the system.

2. **Token Management System (Core)**
   The core system components handle the operations related to token creation, assignment, expiry management, and deletion:
   - **Token Generation** stores tokens in Redis for future assignment.
   - **Token Assignment & Locking** uses Redis’ `SETNX` to ensure tokens are assigned only once.
   - **Token Expiry Management** updates the token expiry time to extend its validity.
   - **Token Unblocking** returns tokens to the available pool.
   - **Token Deletion** removes tokens from Redis when they are no longer needed.

3. **Background Expiry Handler**
   - The **Expiry Manager** scans Redis for expired tokens and deletes them to free up space. This ensures the system does not accumulate expired tokens over time.

#### Architecture Flow

1. **Token Generation**  
   When a client requests a new token (`POST /generate-token`), a new token is created and stored in Redis.

2. **Token Assignment**  
   Clients can request an available token (`POST /assign-token`). If a token is available, it is assigned to the client and locked in Redis using `SETNX`. If no tokens are available, the client receives a `404` error.

3. **Token Expiry Management**  
   Clients can extend the expiry of an assigned token (`POST /keep-alive`). This updates the token’s expiry time in Redis, ensuring the token remains valid for a longer period.

4. **Token Unblocking**  
   When a token is no longer needed, it can be unblocked (`POST /unblock-token`) and returned to the available pool in Redis for future use.

5. **Token Deletion**  
   Clients can delete a token permanently (`DELETE /delete-token`). This operation removes the token from Redis entirely.

6. **Expiry Management (Background)**  
   The **Expiry Manager** runs in the background, periodically scanning Redis for expired tokens and deleting them. This ensures that expired tokens are efficiently cleaned up from the system.

#### Redis Usage

- **Token Storage:** Tokens are stored in Redis using a combination of **strings** for individual tokens and **sorted sets** for managing token expiry times.
- **Token Locking:** The `SETNX` command is used to lock tokens when they are assigned to prevent conflicts.
- **Expiry Handling:** Tokens are given a TTL (Time-To-Live) in Redis, and expired tokens are managed by the **Expiry Manager**.

#### Future Enhancements

- **API Rate Limiting:** Implement rate limiting to control the frequency of requests, ensuring system stability.
- **Horizontal Scaling:** Redis can be scaled horizontally by partitioning token data across multiple Redis instances.
- **Advanced Expiry Strategies:** Support for sliding windows or soft expiry mechanisms can be added for more flexible token expiry handling.
- **High Availability:** Implement Redis failover mechanisms to ensure high availability and fault tolerance in case of Redis failures.

### **🔹 Architecture Diagram**
![Editor _ Mermaid Chart-2025-03-22-182604](https://github.com/user-attachments/assets/222fcac8-0b3b-42d1-b00a-33aecb878199)

**Diagram Explanation:**

-   The diagram depicts the flow of client requests through the Token Management System. Each request triggers specific actions within the system, from generating tokens to handling expiry and background management.
-   **Token Generation** creates and stores tokens in Redis.
-   **Token Assignment** locks the token using `SETNX`, ensuring that tokens are not reassigned to different clients.
-   **Token Expiry** is managed by the system and extended through the **keep-alive** API.
-   **Expiry Management** ensures that expired tokens are periodically removed from Redis.

**Expiry Manager**
![Editor _ Mermaid Chart-2025-03-22-182432](https://github.com/user-attachments/assets/fef23a83-977a-4f03-9d47-f41485f5377f)


----------

## **Getting Started 🏃‍♂️**

### 1. Clone the Repository

To get started with the Token Management Server, begin by cloning the repository:

```bash
git clone https://github.com/manankarani/token-manager.git
cd token-manager
```

### 2. Install Docker

If you don’t have Docker installed, you can follow the steps below based on your operating system:

#### **For macOS:**

1.  Download Docker from the official website: [Docker Desktop for Mac](https://www.docker.com/products/docker-desktop).
2.  Follow the installation instructions.
3.  Once installed, you can verify Docker is running by typing:
```bash
docker --version 
```

#### **For Windows:**

1.  Download Docker from the official website: [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop).
2.  Follow the installation instructions.
3.  After installation, ensure Docker is running and check the version:
```bash
docker --version 
```

#### **For Linux:**

1.  Install Docker using your package manager. For example, on Ubuntu:    
```bash
sudo apt update
sudo apt install docker.io
```
    
2.  Start Docker:    
```bash
sudo systemctl start docker
sudo systemctl enable docker  
```
    
3.  Verify the installation:  
```bash
docker --version
```
    

### 3. Start the Service Using Docker Compose

The system uses **Docker Compose** to simplify the setup by running both the Redis instance and the Token Management Server in Docker containers.

1.  Make sure Docker and Docker Compose are installed.
2.  Run the following command to start both Redis and the Token Management Server:
```bash
docker-compose up
```
This will start the system and make it available at `http://localhost:8080`.

### 4. Horizontally Scaling the System
To scale the Token Management Server horizontally (i.e., running multiple instances for load balancing), you can modify the `docker-compose.yml` file and increase the number of service replicas.

1.  Edit `docker-compose.yml`:
    
```yaml
version: '3.8'
services:
  redis:
    image: redis:latest
    container_name: redis
    ports:
      - "6379:6379"

  app:
    build: .
    deploy:
      replicas: 3
    ports:
      - "0:8080"  # Map a random available port to 8080 inside the container
    depends_on:
      - redis
    env_file:
      - .env
```
    
2.  After modifying the file, redeploy the service with the following command:    
```bash
docker-compose up --scale app=3
```
    
This will deploy three replicas of the Token Management Server, allowing the system to handle a larger number of concurrent requests.

-----
#### Conclusion

The Token Management System is designed for high-performance token handling with a strong focus on scalability and concurrency. By utilizing Redis, it ensures fast token assignment, expiry management, and cleanup. This system is easily extendable, with plans for rate limiting, horizontal scaling, and more sophisticated expiry strategies in the future.
