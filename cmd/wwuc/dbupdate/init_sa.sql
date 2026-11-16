drop database if exists wwuc;

create database wwuc
    character set utf8mb4
    collate utf8mb4_bin;

create table wwuc.`group`
(
    id                bigint auto_increment comment 'Unique identifier'
        primary key,
    name              varchar(50) collate utf8mb4_bin    not null comment 'group name',
    managers          json                               null comment 'the group managers',
    `system`          tinyint  default 0                 null,
    created_by        bigint                             not null comment 'Creator ID',
    created_at        datetime default CURRENT_TIMESTAMP not null comment 'Creation timestamp',
    updated_by        bigint                             null comment 'Updater ID',
    updated_at        datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    deleted_unique_id bigint   default 0                 not null comment 'Deletion flag. 0 indicates not deleted, 1 indicates deleted',
    deleted_at        datetime                           null comment 'Deletion timestamp',
    constraint uk_name
        unique (name, deleted_unique_id)
)
    comment 'group table' charset = utf8mb4;

create index idx_del_id
    on wwuc.`group` (deleted_unique_id, id);

create table wwuc.permission
(
    id                  bigint auto_increment comment 'Unique identifier'
        primary key,
    domain              varchar(50) default 'wsi'             not null,
    code                bigint      default 0                 not null comment 'code in fe',
    type                tinyint(1)  default 0                 not null comment 'Resource type,0:be,1,2:fe',
    name                varchar(50) collate utf8mb4_bin       not null comment 'Resource name',
    path                varchar(256) collate utf8mb4_bin      not null comment 'Resource URL',
    description         varchar(50) collate utf8mb4_bin       null comment 'description for permission',
    dependence_children json                                  null comment 'dependence_children',
    dependence_parents  bigint                                null comment 'dependence_parents',
    create_by           bigint                                null comment 'Creator ID',
    create_time         datetime    default CURRENT_TIMESTAMP not null comment 'Creation timestamp',
    update_by           bigint                                null comment 'Updater ID',
    update_time         datetime    default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    deleted_unique_id   bigint      default 0                 not null comment 'Deletion flag. 0 indicates not deleted, 1 indicates deleted',
    delete_time         datetime                              null comment 'Deletion timestamp',
    constraint k_permission_id
        unique (code)
)
    comment 'role permission' charset = utf8mb4;

create index idx_del_code
    on wwuc.permission (deleted_unique_id, code);

create index idx_del_id
    on wwuc.permission (deleted_unique_id, id);

create index permission_id_index
    on wwuc.permission (id);

create table wwuc.ref_role_permission
(
    id                bigint auto_increment comment 'Unique identifier'
        primary key,
    role_id           bigint                             not null comment 'Role ID',
    permission_id     bigint                             not null comment 'Permission ID',
    created_by        bigint                             null comment 'Creator ID',
    created_at        datetime default CURRENT_TIMESTAMP null comment 'Creation timestamp',
    updated_by        bigint                             null comment 'Updater ID',
    updated_at        datetime default CURRENT_TIMESTAMP null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    deleted_unique_id bigint   default 0                 not null comment 'Deletion flag. 0 indicates not deleted, 1 indicates deleted',
    deleted_at        datetime                           null comment 'Deletion timestamp',
    constraint uk_role_id_permission_id
        unique (role_id, permission_id, deleted_unique_id)
)
    comment 'ref role permission table' charset = utf8mb4;

create index idx_del_group
    on wwuc.ref_role_permission (deleted_unique_id, role_id);

create table wwuc.ref_user_group
(
    id                bigint auto_increment comment 'Unique identifier'
        primary key,
    user_id           bigint                             not null comment 'User ID',
    group_id          bigint                             not null comment 'Group ID',
    created_time      datetime default CURRENT_TIMESTAMP not null comment 'Creation timestamp',
    created_by        bigint                             not null comment 'Creator ID',
    updated_time      datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    updated_by        bigint                             null comment 'Updater ID',
    deleted_unique_id bigint   default 0                 not null comment 'Deletion flag. 0 indicates not deleted, 1 indicates deleted',
    deleted_at        datetime                           null comment 'delete timestamp',
    constraint uk_user_id_group_deleted_unique_id
        unique (deleted_unique_id, user_id, group_id)
)
    comment 'User group ref table' charset = utf8mb4;

create index idx_del_group
    on wwuc.ref_user_group (deleted_unique_id, group_id);

create index idx_del_id
    on wwuc.ref_user_group (deleted_unique_id, id);

create index idx_del_usr
    on wwuc.ref_user_group (deleted_unique_id, user_id);

create table wwuc.ref_user_role
(
    id                bigint auto_increment comment 'Unique identifier'
        primary key,
    user_id           bigint                             not null comment 'User ID',
    role_id           bigint                             not null comment 'Role ID',
    group_id          bigint                             null comment 'user role in group',
    created_by        bigint                             not null comment 'Creator ID',
    created_at        datetime default CURRENT_TIMESTAMP not null comment 'Creation timestamp',
    updated_by        bigint                             null comment 'Updater ID',
    updated_at        datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    deleted_at        datetime                           null comment 'delete timestamp',
    deleted_unique_id bigint   default 0                 not null comment 'del unique key ',
    constraint uk_user_role_del_unique_key
        unique (user_id, group_id, deleted_unique_id, role_id)
)
    comment 'User role ref table' charset = utf8mb4;

create index idx_del_id
    on wwuc.ref_user_role (deleted_unique_id, id);

create index idx_del_role
    on wwuc.ref_user_role (deleted_unique_id, role_id);

create index idx_del_user
    on wwuc.ref_user_role (deleted_unique_id, user_id);

create table wwuc.role
(
    id                bigint auto_increment comment 'Unique identifier'
        primary key,
    name              varchar(50) collate utf8mb4_bin    not null comment 'Role name',
    group_id          bigint                             not null comment 'role in which group',
    `system`          tinyint  default 0                 not null comment 'is system',
    created_by        bigint                             null comment 'Creator ID',
    created_at        datetime default CURRENT_TIMESTAMP not null comment 'Creation timestamp',
    updated_by        bigint                             null comment 'Updater ID',
    updated_at        datetime default CURRENT_TIMESTAMP null on update CURRENT_TIMESTAMP comment 'Update timestamp',
    deleted_unique_id bigint   default 0                 not null comment 'Deletion flag. 0 indicates not deleted, 1 indicates deleted',
    deleted_at        datetime                           null comment 'Deletion timestamp',
    constraint uk_name_group_id
        unique (name, group_id, deleted_unique_id)
)
    comment 'User role' charset = utf8mb4;

create index idx_del_group
    on wwuc.role (deleted_unique_id, group_id);

create index idx_del_id
    on wwuc.role (deleted_unique_id, id);

create index idx_system
    on wwuc.role (`system`);

create table wwuc.user
(
    id                bigint auto_increment comment 'primary key'
        primary key,
    username          varchar(64)                           null comment 'login username',
    type              tinyint     default 1                 not null,
    password          varchar(64) default ''                not null comment 'encode password',
    nickname          varchar(50)                           null comment 'user nickname',
    created_at        datetime    default CURRENT_TIMESTAMP not null comment 'user created time',
    created_by        bigint      default 0                 null comment 'the person create ',
    updated_at        datetime                              null on update CURRENT_TIMESTAMP comment 'user updated time',
    updated_by        bigint      default 0                 null comment 'the person create ',
    last_login_at     datetime                              null comment 'last login time',
    deleted_unique_id bigint      default 0                 null comment 'this user is deleted default 0',
    deleted_at        datetime                              null comment 'user delete time',
    provider          varchar(64)                           null comment 'user provider',
    constraint uniq_username
        unique (deleted_unique_id, username)
)
    comment 'User table' charset = utf8mb4;

create index idx_del_id
    on wwuc.user (deleted_unique_id, id);

create index idx_deleted_unique_id
    on wwuc.user (deleted_unique_id);

create index idx_user
    on wwuc.user (deleted_unique_id, type);

create index idx_username
    on wwuc.user (username);

create table wwuc.user_token
(
    id         bigint auto_increment comment 'Primary key'
        primary key,
    token      varchar(255)                       not null comment 'token',
    token_type enum ('API', 'GATEWAY')            not null comment 'token type',
    expire_at  datetime                           null comment 'token expire datetime',
    created_at datetime default CURRENT_TIMESTAMP null comment 'Create time',
    created_by bigint                             null comment 'Creator id',
    updated_at datetime default CURRENT_TIMESTAMP null on update CURRENT_TIMESTAMP comment 'Update time',
    updated_by bigint                             null comment 'Modify person id'
)
    charset = utf8mb4;