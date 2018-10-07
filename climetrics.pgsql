--
-- PostgreSQL database dump
--

-- Dumped from database version 10.5
-- Dumped by pg_dump version 10.5

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: authentication_role; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.authentication_role AS ENUM (
    'admin',
    'member',
    'revoked'
);


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: authentication; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.authentication (
    username character varying(36) NOT NULL,
    email character varying(254) NOT NULL,
    password character(64) NOT NULL,
    role public.authentication_role DEFAULT 'member'::public.authentication_role NOT NULL,
    user_id uuid NOT NULL
);


--
-- Name: diagnostics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.diagnostics (
    id uuid NOT NULL,
    username character varying(254) NOT NULL,
    report text NOT NULL,
    timestamp_db timestamp with time zone NOT NULL,
    sync_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "timestamp" text NOT NULL
);


--
-- Name: COLUMN diagnostics.sync_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.diagnostics.sync_time IS 'original timestamp as received from the user';


--
-- Name: geolocation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.geolocation (
    ip inet NOT NULL,
    cache json NOT NULL,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: http_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.http_sessions (
    id bigint NOT NULL,
    key bytea,
    data bytea,
    created_on timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    modified_on timestamp with time zone,
    expires_on timestamp with time zone
);


--
-- Name: http_sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.http_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: http_sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.http_sessions_id_seq OWNED BY public.http_sessions.id;


--
-- Name: metrics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.metrics (
    id uuid NOT NULL,
    type character varying(100) NOT NULL,
    text text NOT NULL,
    tags json NOT NULL,
    extra json NOT NULL,
    pid character varying(50) NOT NULL,
    sid uuid NOT NULL,
    "timestamp" text NOT NULL,
    version character varying(20) NOT NULL,
    os character varying(20) NOT NULL,
    arch character varying(20) NOT NULL,
    sync_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    request_id uuid NOT NULL,
    sync_ip inet NOT NULL,
    sync_location json,
    timestamp_db timestamp with time zone NOT NULL
);


--
-- Name: http_sessions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.http_sessions ALTER COLUMN id SET DEFAULT nextval('public.http_sessions_id_seq'::regclass);


--
-- Name: authentication authentication_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authentication
    ADD CONSTRAINT authentication_email_key UNIQUE (email);


--
-- Name: authentication authentication_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authentication
    ADD CONSTRAINT authentication_pkey PRIMARY KEY (username);


--
-- Name: authentication authentication_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authentication
    ADD CONSTRAINT authentication_user_id_key UNIQUE (user_id);


--
-- Name: diagnostics diagnostics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.diagnostics
    ADD CONSTRAINT diagnostics_pkey PRIMARY KEY (id);


--
-- Name: geolocation geolocation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geolocation
    ADD CONSTRAINT geolocation_pkey PRIMARY KEY (ip);


--
-- Name: http_sessions http_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.http_sessions
    ADD CONSTRAINT http_sessions_pkey PRIMARY KEY (id);


--
-- Name: metrics metrics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.metrics
    ADD CONSTRAINT metrics_pkey PRIMARY KEY (id);


--
-- Name: diagnostics_emailx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX diagnostics_emailx ON public.diagnostics USING btree (username);


--
-- Name: metrics_request_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX metrics_request_idx ON public.metrics USING btree (request_id);


--
-- PostgreSQL database dump complete
--

