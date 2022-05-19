create table nsi_field (
    id uuid not null default gen_random_uuid() primary key,
    field_name text not null,
    field_type text not null,
    description text,
    is_domain boolean not null
);

create table domain (
    id uuid not null default gen_random_uuid() primary key,
    nsi_field_id uuid not null,
    domain_value text not null,
    data_type text not null,
    constraint fk_domain_nsi_field
        foreign key(nsi_field_id)
            references nsi_field(id)
);

create table schema_field (
    id uuid not null default gen_random_uuid() primary key,
    nsi_field_id uuid not null,
    constraint fk_schema_field_nsi_field
        foreign key(nsi_field_id)
            references nsi_field(id)
);

create table nsi_schema (
    id uuid not null default gen_random_uuid() primary key,
    name text not null,
    version text not null,
    note text
);

create table quality (
    id uuid not null default gen_random_uuid() primary key,
    value text not null,
    description text
);

create table access (
    id uuid not null default gen_random_uuid() primary key,
    access_group text not null,
    role text not null,
    permission text not null
);

create table dataset (
    id uuid not null default gen_random_uuid() primary key,
    name text not null,
    version text not null,
    nsi_schema_id uuid not null,
    table_name text not null,
    shape geometry not null,
    description text,
    purpose text,
    date_created date not null default current_date,
    created_by text not null,
    quality_id uuid not null,
    constraint fk_dataset_quality
        foreign key(quality_id)
            references quality(id)
);
