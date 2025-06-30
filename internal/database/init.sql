create table if not exists users
(
    id                          bigint generated always as identity,
    tg_id                       bigint unique,
    username                    text,
    first_name                  text,
    last_name                   text,
    language_code               text,
    is_bot                      boolean,
    can_join_groups             boolean,
    can_read_all_group_messages boolean,
    supports_inline_queries     boolean,
    phone                       text,
    created_at                  timestamp default now(),
    primary key (id)
);

create table if not exists chat
(
    id       bigint generated always as identity,
    tg_id    bigint,
    title    varchar,
    owner_id bigint,
    primary key (id),
    constraint fk_owner foreign key (owner_id) references users (id)
);

create table if not exists tournament
(
    id                 bigint generated always as identity,
    created_at         timestamp default now(),
    title              varchar,
    start_date         timestamp,
    end_date           timestamp,
    registration_start timestamp,
    registration_end   timestamp,
    is_active          bool      default false,
    registration_link  varchar,
    tg_channel_link    varchar,
    chat_id            bigint,
    primary key (id),
    constraint fk_chat foreign key (chat_id) references chat (id)
);

create table if not exists team
(
    id           bigint generated always as identity,
    created_at   timestamp default now(),
    name         varchar,
    fastcup_link varchar,
    logo_link    varchar,
    primary key (id)
);

create table if not exists member
(
    id           bigint generated always as identity,
    team_id      bigint,
    name         varchar,
    surname      varchar,
    grad_year    int2,
    role         varchar,
    phone_number varchar,
    primary key (id),
    constraint fk_team foreign key (team_id) references team (id)
);

create table if not exists submission
(
    id                  bigint generated always as identity,
    created_at          timestamp default now(),
    external_id         varchar,
    tournament_id       bigint not null,
    team_id             bigint not null,
    submitted_at        timestamp,
    submission_ip       varchar,
    submission_url      varchar,
    submission_edit_url varchar,
    last_updated_at     timestamp,
    primary key (id),
    constraint fk_team foreign key (team_id) references team (id),
    constraint fk_tournament foreign key (tournament_id) references tournament (id)
);
