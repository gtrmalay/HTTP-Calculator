-- USERS SEQUENCE
CREATE SEQUENCE IF NOT EXISTS public.users_id_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

-- USERS TABLE
CREATE TABLE IF NOT EXISTS public.users (
    id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    login varchar(255) NOT NULL,
    password_hash varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_login_key UNIQUE (login)
);

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

-- EXPRESSIONS SEQUENCE
CREATE SEQUENCE IF NOT EXISTS public.expressions_id_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

-- EXPRESSIONS TABLE
CREATE TABLE IF NOT EXISTS public.expressions (
    id integer NOT NULL DEFAULT nextval('expressions_id_seq'::regclass),
    user_id integer,
    expression text NOT NULL,
    result double precision,
    status varchar(50) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT expressions_pkey PRIMARY KEY (id),
    CONSTRAINT expressions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users (id)
);

ALTER SEQUENCE public.expressions_id_seq OWNED BY public.expressions.id;

-- TASKS TABLE
CREATE TABLE IF NOT EXISTS public.tasks (
    id varchar(36) NOT NULL,
    expression_id integer NOT NULL,
    arg1 varchar(255) NOT NULL,
    arg2 varchar(255) NOT NULL,
    operation varchar(10) NOT NULL,
    operation_time integer NOT NULL,
    status varchar(50) NOT NULL,
    result double precision,
    depends_on text[],
    CONSTRAINT tasks_pkey PRIMARY KEY (id),
    CONSTRAINT tasks_expression_id_fkey FOREIGN KEY (expression_id) REFERENCES public.expressions (id)
);

-- TASK_QUEUE TABLE
CREATE TABLE IF NOT EXISTS public.task_queue (
    task_id varchar(36) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT task_queue_pkey PRIMARY KEY (task_id),
    CONSTRAINT task_queue_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks (id)
);
