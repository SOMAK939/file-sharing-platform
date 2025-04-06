


## Features Implemented

### 1. **User Authentication (JWT-Based)**
   - Implemented **JWT (JSON Web Token)** authentication to secure user access.
   - Used `golang-jwt` package to generate and verify tokens.
   - Stored hashed passwords in PostgreSQL using `bcrypt`.

   
### 2. **File Upload & Storage**
   - Allowed users to upload files via REST API.
   - Stored metadata (filename, size, upload time, etc.) in PostgreSQL.
   - Used AWS S3 for storage (with a local storage option for development/testing).
   - In order to allow public access or shareable URLs I changed the bucket policy.
    

### 3. **File Download & Retrieval**
   - Provided API endpoints to fetch file metadata and retrieve files.
   -Allowed public access to the bucket's objects----(not considered a best practice)
   - Enabled range-based retrieval for efficient large file downloads.

### 4. **Caching for Performance Optimization**
   - Implemented Redis using WSL
   - Used Redis to cache frequently accessed data (e.g., user info, file metadata).
   

### 5. **Rate Limiting & Security**
   
   - Implemented IP-based rate limiting with `golang.org/x/time/rate`.

### 6. **Concurrent Processing with Goroutines**
   - Used Goroutines and channels for handling large file uploads efficiently.
   - Optimized request handling using worker pools.

### 7. **Database Management with PostgreSQL**
   - Designed a structured database schema for file metadata storage.
   - Used `gorm` as the ORM for database interactions.
   - Implemented transactional operations to ensure data consistency.

---

## Challenges Faced 

### 1. **Setting Up PostgreSQL on AWS EC2**
   - Faced issues with installing and initializing PostgreSQL on **Amazon Linux**.
   

### 2. **AWS S3 Permissions & Upload Issues**
   - Encountered permission errors when trying to upload files to S3.
   - Solution: Configured correct IAM roles and bucket policies.

### 3. **Handling Large File Uploads**
   - Standard file uploads were inefficient for large files.
   - Solution: Implemented multi-part upload using AWS SDK and Goroutines.

### 4. **Caching Metadata Efficiently**
   - Initial queries to PostgreSQL were slow for frequently accessed files.
   - Solution: Added Redis caching to speed up repeated queries.




### 5.**Docker**
- Docker was not properly configured.
- Difficulty faced: With the golang version installation.
- My go.mod file required Go 1.24.1, but my Docker image was using Go 1.21.13.
- After fixing the above issue Go app was still failing to connect to the database.
---

### 6.**Connecting to EC2 via CLI**
- Difficulty faced:
go: go.mod requires go >= 1.24.1 (running go 1.24.0; GOTOOLCHAIN=local)

--Fixed it by installing Go1.24.1 .System was using Go1.24.0
--app binary was missing





## Future Improvements
- Implementation of a WebSocket-based real-time notifications for upload completion.

- Deployment of  a fully containerized version using Docker.

- Deployment over EC2 via CLI


Note:
The uploads folder contains all the files uploaded during testing of the routes.









