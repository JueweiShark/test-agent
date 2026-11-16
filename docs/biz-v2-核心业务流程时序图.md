# WWUC v2 核心业务流程时序图

> **版本**: v1.0.0
> **日期**: 2026-08-07
> **配套文档**: [biz-v2-开发文档.md](file:///Users/zzj/go/src/wwuc/docs/biz-v2-开发文档.md)
> **范围**: 选取 v2 biz 层最能体现业务复杂度的 6 个核心流程,以 Mermaid 时序图呈现
> **图例约定**:
>
> - 实线箭头 `→` 表示方法调用
> - 虚线箭头 `-->>` 表示返回
> - `alt`/`opt`/`loop` 为分支/可选/循环
> - 右侧注释 `Note` 标注关键决策点与错误码

***

## 目录

- [流程一:LoginV2 登录主流程](#流程一loginv2-登录主流程)
- [流程二:权限继承读取(3 表 join)](#流程二权限继承读取3-表-join)
- [流程三:SetGroupPermissions 设置企业权限范围](#流程三setgrouppermissions-设置企业权限范围)
- [流程四:CreateRole 创建角色](#流程四createrole-创建角色)
- [流程五:AssignRoleToUser 为企业加员工](#流程五assignroletouser-为企业加员工)
- [流程六:UpdateGroupEmployee 修改企业员工](#流程六updategroupemployee-修改企业员工)
- [附录:核心流程事务边界速查](#附录核心流程事务边界速查)

***

## 流程一:LoginV2 登录主流程

**场景**: 用户输入用户名密码登录,系统校验身份、签发 token、聚合用户在企业中的角色与有效权限。
**事务边界**: 无显式事务。token 持久化是单表 insert;`UpdateLastLoginAt` 失败仅记日志,不阻塞登录。
**核心难点**: 超管分支 + 普通员工的 3 表 join 权限计算 + 按 groupId 聚合角色(避免 N+1)。

```mermaid
sequenceDiagram
    autonumber
    participant FE as 前端
    participant SVC as Service<br/>(service/v2)
    participant UC as AuthUsecase<br/>(biz/v2)
    participant UR as UserRepo
    participant RUR as RefUserRoleRepo
    participant PR as PermissionRepo
    participant GR as GroupRepo
    participant RR as RoleRepo
    participant TU as TokenUsecase<br/>(biz/user)
    participant UUC as UserUsecase

    FE->>SVC: LoginV2(username, password)
    SVC->>UC: LoginV2(ctx, username, password)

    rect rgb(245, 245, 245)
        Note over UC,UR: 步骤 2-3: 用户校验
        UC->>UR: FindByUserName(username)
        UR-->>UC: u, err
        alt err != nil || u == nil
            UC-->>SVC: errUserNotExist (401)
        end
        alt u.Status == 0
            UC-->>SVC: errUseIsForbidden (403)
        end
    end

    rect rgb(255, 250, 240)
        Note over UC: 步骤 4: 密码比对 (bcrypt)
        UC->>UC: bcrypt.CompareHashAndPassword<br/>(u.Password, password)
        alt 不匹配
            UC-->>SVC: errUserPWDNotMatch (401)
        end
    end

    rect rgb(240, 248, 255)
        Note over UC,TU: 步骤 5: 签发 token (JWT 只装 userId + exp)
        UC->>TU: GenerateToken(userId, "API", &exp)
        TU-->>UC: token, expTime
        alt 签发失败
            UC-->>SVC: errUserTokenGenerateFail (500)
        end
    end

    rect rgb(248, 255, 240)
        Note over UC,RUR: 步骤 6: 查用户角色绑定 [{roleId, groupId}]
        UC->>RUR: FindByUserId(userId)
        RUR-->>UC: refs
        UC->>UC: 去重 roleIds<br/>IsSuperAdmin(roleIds)?
    end

    alt 超管 (roleIds 含 roleAdminId=1)
        rect rgb(255, 245, 238)
            Note over UC,PR: 超管分支: groups=[], permissions=FindAll()
            UC->>PR: FindAll()
            PR-->>UC: allPerms
            UC->>UC: groups = []*GroupGroup{}
            Note over UC: isSuperAdmin = true
        end
    else 普通员工
        rect rgb(238, 245, 255)
            alt len(roleIds) == 0
                UC-->>SVC: errUserNoGroup (403)<br/>未关联任何企业
            end
            Note over UC,PR: 步骤 8: 有效权限 = 角色权限 ∩ 企业 scope (3表join)
            UC->>PR: FindEffectiveByUserId(userId)
            PR-->>UC: perms

            Note over UC: 步骤 9: 按 groupId 聚合 refs, 去重 groupIds
            UC->>UC: assembleGroups(groupIds, refs)
            UC->>GR: FindByIds(groupIds)
            GR-->>UC: groups
            UC->>RR: FindByIds(allRoleIds)
            RR-->>UC: roles
            UC->>UC: 按 groupId 聚合 roles<br/>跳过已软删的 group
            Note over UC: isSuperAdmin = false
        end
    end

    rect rgb(245, 245, 255)
        Note over UC,UUC: 步骤 10: 更新最后登录时间 (失败仅记日志, 不阻塞)
        UC->>UUC: UpdateLastLoginAt(userId)
        UUC->>UR: UpdateLastLoginAt(userId)
        UR-->>UUC: err?
        opt err != nil
            UUC->>UUC: log.Errorf(...)<br/>不返回错误
        end
    end

    UC-->>SVC: LoginResult{Token, ExpireTime, User,<br/>Groups, Permissions, IsSuperAdmin}
    SVC->>SVC: convert → proto.LoginV2Reply
    SVC-->>FE: 200 OK + LoginV2Reply
```

**返回结构**:

```go
type LoginResult struct {
    Token        string
    ExpireTime   int64           // unix 秒
    User         *User
    Groups       []*GroupGroup   // 超管:[]; 普通:按企业聚合的角色
    Permissions  []*Permission   // 超管:全部; 普通:有效权限
    IsSuperAdmin bool
}
```

***

## 流程二:权限继承读取(3 表 join)

**场景**: 登录后 / 总览时,查某用户的有效权限。这是权限继承模型的**读取侧**。
**核心 SQL**: `ref_user_role ⨝ ref_role_permission ⨝ ref_group_permission ⨝ permission`
**语义**: 用户有效权限 = 角色权限 ∩ 企业 scope(父域约束子集)

```mermaid
sequenceDiagram
    autonumber
    participant UC as AuthUsecase<br/>/ RoleUsecase
    participant PR as PermissionRepo<br/>(biz 接口)
    participant DR as permissionRepo<br/>(data/v2 实现)
    participant DB as MySQL

    UC->>PR: FindEffectiveByUserId(userId)
    Note over PR,DR: 仓储接口契约: 3 表 join 查有效权限

    DR->>DB: SELECT DISTINCT p.*
    Note over DR,DB: FROM ref_user_role ur<br/>JOIN ref_role_permission rrp<br/>  ON ur.role_id = rrp.role_id<br/>JOIN ref_group_permission rgp<br/>  ON ur.group_id = rgp.group_id<br/>  AND rrp.permission_id = rgp.permission_id<br/>JOIN permission p<br/>  ON rrp.permission_id = p.id<br/>WHERE ur.user_id = ?<br/>  AND ur.deleted_unique_id = 0<br/>  AND rrp.deleted_unique_id = 0<br/>  AND rgp.deleted_unique_id = 0<br/>  AND p.deleted_unique_id = 0

    DB-->>DR: rows
    DR->>DR: mapper.MapperSlice<br/>(schema → biz.Permission)
    DR-->>UC: []*Permission

    Note over UC: 有效权限 = 角色权限 ∩ 企业 scope
```

**权限继承模型示意**:

```mermaid
graph TB
    subgraph 父域[企业 scope 父域]
        GP1[ref_group_permission<br/>groupId=A, permId=1]
        GP2[ref_group_permission<br/>groupId=A, permId=2]
        GP3[ref_group_permission<br/>groupId=A, permId=3]
    end

    subgraph 子集[角色权限 子集]
        RP1[ref_role_permission<br/>roleId=R, permId=1]
        RP2[ref_role_permission<br/>roleId=R, permId=2]
        RPX[ref_role_permission<br/>roleId=R, permId=9 ❌越界]
    end

    subgraph 绑定[用户角色绑定]
        UR[ref_user_role<br/>userId=U, roleId=R, groupId=A]
    end

    UR --> RP1
    UR --> RP2
    UR --> RPX
    RP1 -.⊆.-> GP1
    RP2 -.⊆.-> GP2
    RPX -.✗ 不在 scope.-> 父域

    subgraph 结果[用户有效权限]
        EFF[permId=1, permId=2<br/>= 角色权限 ∩ 企业 scope]
    end

    RP1 --> EFF
    RP2 --> EFF
```

> **写入侧维护**: 当企业 scope 收缩移除 permId=2 时,`SetGroupPermissions` 会级联软删对应的 `ref_role_permission`,保证读取侧不会返回脏数据。详见 [流程三](#流程三setgrouppermissions-设置企业权限范围)。

***

## 流程三:SetGroupPermissions 设置企业权限范围

**场景**: 超管配置某企业的权限范围(父域),采用**全量覆盖**语义。
**事务边界**: ✅ 必须事务。涉及 diff 计算 + 软删移除 + 级联软删越界角色权限 + 新增,必须原子。
**核心难点**: 权限继承模型的**写入侧维护**——企业 scope 收缩时,必须级联软删越界的 `ref_role_permission`。

```mermaid
sequenceDiagram
    autonumber
    participant SVC as Service
    participant UC as GroupUsecase
    participant GR as GroupRepo
    participant RGPR as RefGroupPermissionRepo
    participant TM as Transaction<br/>(common)

    SVC->>UC: SetGroupPermissions(ctx, userId,<br/>groupId, permissionIds)

    rect rgb(245, 245, 245)
        Note over UC,GR: 步骤 1: 校验企业存在性
        UC->>GR: FindById(groupId)
        GR-->>UC: group, err
        alt err != nil
            UC-->>SVC: errGroupNotFound (404)
        end
    end

    rect rgb(255, 250, 240)
        Note over UC,RGPR: 步骤 2: 查旧 scope
        UC->>RGPR: FindByGroupId(groupId)
        RGPR-->>UC: existing (旧 ref_group_permission)
    end

    rect rgb(248, 255, 240)
        Note over UC: 步骤 3: diff 计算 (事务外)
        UC->>UC: existingSet = {旧 permIds}<br/>targetSet = {新 permIds}
        UC->>UC: removedIds = 旧scope - 新scope<br/>addedIds = 新scope - 旧scope
        Note over UC: removedIds 需级联软删角色权限<br/>addedIds 需新增 ref_group_permission
    end

    rect rgb(240, 248, 255)
        Note over UC,TM: 步骤 4: 事务内执行 (原子)
        UC->>TM: InTx(ctx, func)
        TM->>TM: BEGIN TX

        rect rgb(255, 245, 238)
            Note over UC,RGPR: 4a: 软删移除的 ref_group_permission
            loop 每个 removed permId
                UC->>RGPR: DeleteByGroupIdAndPermissionId<br/>(groupId, permId, userId)
                RGPR-->>UC: ok
            end
        end

        rect rgb(255, 240, 245)
            Note over UC,RGPR: 4b: 级联软删越界的 ref_role_permission (核心!)
            opt len(removedIds) > 0
                UC->>RGPR: CascadeDeleteRolePermissions<br/>(groupId, removedIds, userId)
                Note over RGPR: SQL: 软删 ref_role_permission<br/>WHERE role_id IN (<br/>  SELECT id FROM role<br/>  WHERE group_id=?<br/>  AND deleted_unique_id=0<br/>)<br/>AND perm_id IN (removedIds)
                RGPR-->>UC: ok
                Note over UC: 保证读取侧 (3表join)<br/>不再返回越界权限
            end
        end

        rect rgb(240, 255, 240)
            Note over UC,RGPR: 4c: 新增 added 的 ref_group_permission
            loop 每个 added permId
                UC->>RGPR: Create(RefGroupPermission{<br/>groupId, permId, createdBy=userId})
                RGPR-->>UC: ok
            end
        end

        alt 全部成功
            TM->>TM: COMMIT
            TM-->>UC: nil
        else 任一失败
            TM->>TM: ROLLBACK
            TM-->>UC: err
            UC-->>SVC: errInternalError (500)
        end
    end

    UC-->>SVC: nil
```

**diff 计算示例**:

```
旧 scope = {1, 2, 3, 4}
新 scope = {2, 3, 5, 6}
removedIds = {1, 4}   → 软删 ref_group_permission + 级联软删 ref_role_permission
addedIds   = {5, 6}   → 新增 ref_group_permission
```

***

## 流程四:CreateRole 创建角色

**场景**: 超管为某企业创建角色并绑定初始权限。
**事务边界**: ✅ 必须事务。insert → 取 id → 生成 code → update code → 绑权限,必须原子。
**核心规范**: **scope 校验必须在事务外**,避免 403 被事务回滚遮蔽为 500。

```mermaid
sequenceDiagram
    autonumber
    participant SVC as Service
    participant UC as RoleUsecase
    participant RR as RoleRepo
    participant RGPR as RefGroupPermissionRepo
    participant RPR as RefRolePermissionRepo
    participant TM as Transaction

    SVC->>UC: CreateRole(ctx, r, permIds)

    rect rgb(245, 245, 245)
        Note over UC,RGPR: 事务外前置校验 (避免 403 被遮蔽为 500)
        UC->>RR: ExistsByName(groupId, name)
        RR-->>UC: ok
        alt 角色名重复
            UC-->>SVC: errRoleNameExists (409)
        end

        UC->>UC: dedup = dedupPermissionIds(permIds)<br/>(去重 + 过滤 <=0)

        UC->>RGPR: FindByGroupId(groupId)
        RGPR-->>UC: groupPerms (企业 scope)
        UC->>UC: scopeSet = {企业 permIds}
        loop 每个 dedup permId
            alt permId ∉ scopeSet
                UC-->>SVC: errPermissionOutOfScope (403)
                Note over UC: 403 直接返回, 不进事务
            end
        end
    end

    rect rgb(240, 248, 255)
        Note over UC,TM: 事务内执行 (原子)
        UC->>TM: InTx(ctx, func)
        TM->>TM: BEGIN TX

        rect rgb(255, 250, 240)
            Note over UC,RR: 4a: 插入角色, GORM 回填 r.Id
            UC->>RR: Create(r)
            RR-->>UC: ok (r.Id 已回填)
            alt r.Id == nil
                TM->>TM: ROLLBACK
                UC-->>SVC: errInternalError (500)
            end
        end

        rect rgb(248, 255, 240)
            Note over UC,RR: 4b: 生成 code = role{YYMMdd}{id:05d}
            UC->>UC: code = fmt.Sprintf(<br/>"role%s%05d",<br/>time.Now().Format("060102"), *r.Id)
            Note over UC: 示例: role26080700001
            UC->>RR: UpdateCode(*r.Id, code)
            RR-->>UC: ok
            UC->>UC: r.Code = code (回显)
        end

        rect rgb(240, 255, 240)
            Note over UC,RPR: 4c: 绑定初始权限 (新角色无旧绑定, 直接批量插)
            opt len(dedup) > 0
                UC->>UC: overwriteRolePermissions<br/>(复用 ctx 中的 tx)
                UC->>RPR: DeleteByRoleId(userId, roleId)<br/>(新角色此处为 no-op)
                UC->>RPR: CreateBatch(refs)
                RPR-->>UC: ok
            end
        end

        alt 全部成功
            TM->>TM: COMMIT
            TM-->>UC: nil
        else 任一失败
            TM->>TM: ROLLBACK
            UC->>UC: log.Errorf("create role failed: %v")
            UC-->>SVC: errInternalError (500)
        end
    end

    UC-->>SVC: nil (r.Code 已回填)
```

**code 生成规则**: `role` + `YYMMdd` + 5 位零填充 id,例如 `role26080700001`。

***

## 流程五:AssignRoleToUser 为企业加员工

**场景**: 超管把某用户加入某企业,并一次绑定多个角色。
**事务边界**: ✅ 必须事务。写 `ref_user_group`(成员关系)+ 写多条 `ref_user_role`(角色绑定),必须原子。
**核心校验**: 重复加入校验 + 角色合法性校验(全部属于该企业)。

```mermaid
sequenceDiagram
    autonumber
    participant SVC as Service
    participant UC as RoleUsecase
    participant RUG as RefUserGroupRepo
    participant RR as RoleRepo
    participant RUR as RefUserRoleRepo
    participant TM as Transaction

    SVC->>UC: AssignRoleToUser(ctx, userId, groupId,<br/>roleIds, status, remarks, operatorId)

    rect rgb(245, 245, 245)
        Note over UC,RUG: 事务外前置校验
        UC->>RUG: Exists(userId, groupId)
        RUG-->>UC: ok
        alt 已加入 (未软删)
            UC-->>SVC: errUserAlreadyInGroup (409)
        end

        UC->>UC: dedup = dedupPermissionIds(roleIds)<br/>(复用: 去重 + 过滤 <=0)

        Note over UC,RR: 校验 roleIds 全部属于该企业
        UC->>RR: FindByIds(dedup)
        RR-->>UC: roles
        alt len(roles) != len(dedup)
            UC-->>SVC: errRoleNotFound (404)
        end
        loop 每个 role
            alt role.GroupId != groupId
                UC-->>SVC: errRoleNotInGroup (403)
            end
        end

        UC->>UC: status <= 0 ? status = 1 : status
    end

    rect rgb(240, 248, 255)
        Note over UC,TM: 事务内执行 (原子)
        UC->>TM: InTx(ctx, func)
        TM->>TM: BEGIN TX

        rect rgb(255, 250, 240)
            Note over UC,RUG: 5a: 写 ref_user_group (成员关系)
            UC->>RUG: Create(RefUserGroup{<br/>UserId, GroupId, Status,<br/>Remarks, CreatedBy=operatorId})
            RUG-->>UC: ok
        end

        rect rgb(240, 255, 240)
            Note over UC,RUR: 5b: 写多条 ref_user_role (角色绑定)
            loop 每个 rid in dedup
                UC->>RUR: Create(RefUserRole{<br/>UserId, RoleId=rid,<br/>GroupId, CreateBy=operatorId})
                RUR-->>UC: ok
            end
        end

        alt 全部成功
            TM->>TM: COMMIT
            TM-->>UC: nil
        else 任一失败
            TM->>TM: ROLLBACK
            UC->>UC: log.Errorf("assign role to user failed")
            UC-->>SVC: errInternalError (500)
        end
    end

    UC-->>SVC: nil
```

**关系表语义**:

```
ref_user_group (成员关系): 一个 (user_id, group_id) 未软删时唯一
    ↓
ref_user_role (角色绑定): 一个 ref_user_group 可对应多条 (多角色)
```

***

## 流程六:UpdateGroupEmployee 修改企业员工

**场景**: 修改某企业员工的角色/状态/备注。角色采用**全量覆盖**语义(空=清空角色,但不删成员关系)。
**事务边界**: ✅ 必须事务。更新 `ref_user_group` + 全量覆盖 `ref_user_role`(先软删旧,再批量插新)。

```mermaid
sequenceDiagram
    autonumber
    participant SVC as Service
    participant UC as RoleUsecase
    participant RUG as RefUserGroupRepo
    participant RR as RoleRepo
    participant RUR as RefUserRoleRepo
    participant TM as Transaction

    SVC->>UC: UpdateGroupEmployee(ctx, userId, groupId,<br/>roleIds, status, remarks, operatorId)

    rect rgb(245, 245, 245)
        Note over UC,RUG: 事务外前置校验
        UC->>RUG: Exists(userId, groupId)
        RUG-->>UC: exist
        alt !exist
            UC-->>SVC: errUserNotInGroup (404)<br/>员工必须已加入企业
        end

        UC->>UC: dedup = dedupPermissionIds(roleIds)
        UC->>RR: FindByIds(dedup)
        RR-->>UC: roles
        alt len(roles) != len(dedup)
            UC-->>SVC: errRoleNotFound (404)
        end
        loop 每个 role
            alt role.GroupId != groupId
                UC-->>SVC: errRoleNotInGroup (403)
            end
        end
    end

    rect rgb(240, 248, 255)
        Note over UC,TM: 事务内执行 (原子)
        UC->>TM: InTx(ctx, func)
        TM->>TM: BEGIN TX

        rect rgb(255, 250, 240)
            Note over UC,RUG: 6a: 更新 ref_user_group (status + remarks)
            UC->>RUG: Update(RefUserGroup{<br/>UserId, GroupId, Status,<br/>Remarks, UpdatedBy=operatorId})
            RUG-->>UC: ok
        end

        rect rgb(255, 240, 245)
            Note over UC,RUR: 6b: 全量覆盖 ref_user_role (先软删旧, 再插新)
            UC->>RUR: FindByUserIdAndGroupId(userId, groupId)
            RUR-->>UC: oldRefs (旧绑定)
            loop 每个 oldRef
                UC->>RUR: Delete(userId, oldRef.RoleId, groupId)
                RUR-->>UC: ok
            end

            Note over UC: 即使 dedup 为空也要执行上面的软删<br/>语义: 空 roleIds = 清空角色 (但不删成员关系)
            loop 每个 rid in dedup
                UC->>RUR: Create(RefUserRole{<br/>UserId, RoleId=rid,<br/>GroupId, CreateBy=operatorId})
                RUR-->>UC: ok
            end
        end

        alt 全部成功
            TM->>TM: COMMIT
            TM-->>UC: nil
        else 任一失败
            TM->>TM: ROLLBACK
            UC->>UC: log.Errorf("update group employee failed")
            UC-->>SVC: errInternalError (500)
        end
    end

    UC-->>SVC: nil
```

**与 AssignRoleToUser 的差异**:

| 维度               | AssignRoleToUser | UpdateGroupEmployee |
| ---------------- | ---------------- | ------------------- |
| 前置               | 重复加入返回 409       | 未加入返回 404           |
| ref\_user\_group | Create           | Update              |
| ref\_user\_role  | 直接 Create        | 先软删旧 + 再 Create     |
| 空角色语义            | —                | 清空角色,保留成员关系         |

***

## 附录:核心流程事务边界速查

| 流程                    | 事务 | scope 校验位置          | 核心难点                                 |
| --------------------- | -- | ------------------- | ------------------------------------ |
| LoginV2               | ❌  | —                   | 超管分支 + 3 表 join + 按 group 聚合(避免 N+1) |
| FindEffectiveByUserId | ❌  | —                   | 3 表 join:角色权限 ∩ 企业 scope             |
| SetGroupPermissions   | ✅  | 事务外                 | diff 计算 + 级联软删越界角色权限                 |
| CreateRole            | ✅  | **事务外**(避免 403→500) | insert→生成 code→update code→绑权限       |
| UpdateRole            | ✅  | 事务外                 | 系统角色特殊处理 + 全量覆盖权限                    |
| DeleteRole            | ✅  | —                   | System=1 不可删 + RoleInUse 校验 + 级联软删   |
| AssignRoleToUser      | ✅  | 事务外                 | 重复加入校验 + 角色合法性校验                     |
| UpdateGroupEmployee   | ✅  | 事务外                 | 全量覆盖角色(空=清空) + 保留成员关系                |

> **事务使用铁律**: 所有业务校验(403/404/409)**必须在事务外**完成,事务内只处理基础设施错误。否则业务错误会被事务回滚遮蔽为 500,丢失错误语义。

***

## 附录:Mermaid 渲染说明

本文档所有时序图使用 Mermaid `sequenceDiagram` 语法。在支持 Mermaid 的 Markdown 查看器(VS Code + Mermaid 插件 / GitHub / 飞书文档)中可直接渲染。若需导出为图片,可使用 [mermaid.live](https://mermaid.live) 在线编辑器粘贴渲染。
