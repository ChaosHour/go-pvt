
\u char_test_db


DROP TABLE IF EXISTS my_table;
DROP TABLE IF EXISTS my_log;
DROP PROCEDURE IF EXISTS my_proc;
DROP EVENT IF EXISTS my_event;
DROP VIEW IF EXISTS my_view;


-- Create a table
CREATE TABLE my_table (
    col1 INT,
    col2 INT
);

/*
-- Create a table for my_log
CREATE TABLE my_log (
    id INT PRIMARY KEY AUTO_INCREMENT,
    message VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
*/

-- Create a table for trigger
CREATE TABLE employees (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    salary INT DEFAULT 3000
);


-- Insert some data
INSERT INTO my_table (col1, col2) VALUES (1, 2), (3, 4), (5, 6);


-- Create a stored procedure
DELIMITER $$
CREATE PROCEDURE my_proc()
BEGIN
    SELECT 'Hello, world!' AS message;
END $$
DELIMITER ;

-- Create an event
CREATE EVENT my_event
ON SCHEDULE EVERY 1 HOUR
DO
    INSERT INTO my_table (col1, col2) VALUES (1, 2);

-- Create a view
CREATE VIEW my_view AS
    SELECT col1, col2 FROM my_table WHERE col1 > 0;

-- Create a trigger
CREATE TRIGGER set_default_salary
BEFORE INSERT ON employees
FOR EACH ROW
SET NEW.salary = 5000;

select sleep(3);


SELECT
    ROUTINE_NAME,
    ROUTINE_TYPE,
    DEFINER
FROM
    INFORMATION_SCHEMA.ROUTINES
WHERE
    ROUTINE_SCHEMA = 'char_test_db'
UNION ALL
SELECT
    TABLE_NAME,
    'VIEW',
    DEFINER
FROM
    INFORMATION_SCHEMA.VIEWS
WHERE
    TABLE_SCHEMA = 'char_test_db'
UNION ALL
SELECT
    TRIGGER_NAME,
    'TRIGGER',
    DEFINER
FROM
    INFORMATION_SCHEMA.TRIGGERS
WHERE
    TRIGGER_SCHEMA = 'char_test_db'
UNION ALL
SELECT
    EVENT_NAME,
    'EVENT',
    DEFINER
FROM
    INFORMATION_SCHEMA.EVENTS
WHERE
    EVENT_SCHEMA = 'char_test_db';