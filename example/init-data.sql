-- Sample data for testing the sqld examples

-- Insert sample users
INSERT INTO users (name, email, age, status, role, country, verified, created_at) VALUES
('John Doe', 'john@example.com', 28, 'active', 'user', 'US', true, '2024-01-15 10:00:00'),
('Jane Smith', 'jane@example.com', 32, 'active', 'admin', 'US', true, '2024-01-20 11:30:00'),
('Bob Johnson', 'bob@example.com', 45, 'active', 'manager', 'CA', true, '2024-02-01 09:15:00'),
('Alice Brown', 'alice@example.com', 26, 'pending', 'user', 'UK', false, '2024-02-15 14:20:00'),
('Charlie Wilson', 'charlie@example.com', 35, 'active', 'user', 'US', true, '2024-03-01 16:45:00'),
('Diana Prince', 'diana@example.com', 29, 'active', 'admin', 'US', true, '2024-03-10 08:30:00'),
('Eve Adams', 'eve@example.com', 22, 'inactive', 'user', 'FR', false, '2024-03-15 12:00:00'),
('Frank Miller', 'frank@example.com', 38, 'active', 'manager', 'DE', true, '2024-04-01 10:30:00'),
('Grace Lee', 'grace@example.com', 31, 'active', 'user', 'JP', true, '2024-04-10 15:20:00'),
('Henry Taylor', 'henry@example.com', 27, 'pending', 'user', 'AU', false, '2024-04-15 13:45:00'),
('Ivy Chen', 'ivy@example.com', 24, 'active', 'user', 'CN', true, '2024-05-01 11:10:00'),
('Jack Davis', 'jack@example.com', 42, 'active', 'admin', 'US', true, '2024-05-05 09:25:00'),
('Kate Anderson', 'kate@example.com', 33, 'active', 'manager', 'CA', true, '2024-05-10 14:40:00'),
('Leo Martinez', 'leo@example.com', 25, 'inactive', 'user', 'ES', false, '2024-05-15 16:15:00'),
('Mia Rodriguez', 'mia@example.com', 30, 'active', 'user', 'MX', true, '2024-06-01 12:30:00');

-- Insert sample posts
INSERT INTO posts (user_id, title, content, published, category, tags, created_at) VALUES
(1, 'Getting Started with Go', 'A comprehensive guide to Go programming...', true, 'programming', ARRAY['go', 'tutorial', 'beginners'], '2024-01-16 10:00:00'),
(1, 'Advanced Go Patterns', 'Exploring advanced patterns in Go development...', true, 'programming', ARRAY['go', 'advanced', 'patterns'], '2024-02-01 11:00:00'),
(2, 'Database Design Best Practices', 'How to design efficient database schemas...', true, 'database', ARRAY['sql', 'design', 'performance'], '2024-01-25 14:30:00'),
(3, 'Project Management Tips', 'Essential tips for managing software projects...', true, 'management', ARRAY['project', 'team', 'productivity'], '2024-02-05 09:15:00'),
(5, 'Web API Security', 'Securing your REST APIs the right way...', true, 'security', ARRAY['api', 'security', 'authentication'], '2024-03-05 15:20:00'),
(6, 'Microservices Architecture', 'Building scalable microservices...', true, 'architecture', ARRAY['microservices', 'scalability', 'docker'], '2024-03-12 10:45:00'),
(8, 'DevOps Best Practices', 'Streamlining your deployment pipeline...', true, 'devops', ARRAY['cicd', 'deployment', 'automation'], '2024-04-03 13:30:00'),
(9, 'Frontend Frameworks Comparison', 'React vs Vue vs Angular in 2024...', false, 'frontend', ARRAY['react', 'vue', 'angular'], '2024-04-12 16:00:00'),
(12, 'Database Optimization', 'Techniques for optimizing database performance...', true, 'database', ARRAY['optimization', 'indexing', 'queries'], '2024-05-07 11:20:00'),
(13, 'Team Leadership', 'Building and leading effective development teams...', true, 'management', ARRAY['leadership', 'team', 'communication'], '2024-05-12 14:15:00');

-- Add some users with deleted_at for testing soft deletes
UPDATE users SET deleted_at = '2024-06-01 10:00:00' WHERE email = 'eve@example.com';
UPDATE users SET deleted_at = '2024-06-05 15:30:00' WHERE email = 'leo@example.com';