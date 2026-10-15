# CamoPanel 应用商店规范与开发指南 (App Store Specification)

本文档定义了 CamoPanel 应用商店的应用包结构、配置文件规范以及与 Docker Compose 的集成标准。第三方开发者或社区贡献者可遵循本指南为 CamoPanel 制作并分发自托管应用模板。

---

## 1. 目录结构规范

每个应用必须作为一个独立文件夹存放于应用仓库中（例如 `apps/<app_id>/` 或外部自定义应用目录 `<data_dir>/custom_apps/<app_id>/`）。

```text
apps/<app-id>/
├── app.yaml              # [必需] 应用元数据、受控目录、用户表单配置
├── compose.yaml          # [必需] Docker Compose 标准编排模板
├── icon.svg (或 icon.png) # [可选] 应用独立图标（亦可在 app.yaml 中使用内置图标键名）
└── README.md             # [可选] 应用使用指引、初始账号说明或特别注意事项
```

---

## 2. 配置文件 (`app.yaml`) 规范

`app.yaml` 用于定义应用的元数据、前端表单渲染规则、安全约束声明及网关反代属性。

### 2.1 完整字段 Schema

```yaml
# 规范版本号，当前为 "1.0"
schema_version: "1.0"

# 基础元数据
id: string               # [必需] 唯一应用标识，只能包含字母、数字、点、下划线与中划线（2-63字符）
name: string             # [必需] 人类可读的应用展示名称
category: string         # [必需] 分类：web | database | devops | tools | media 等
version: string          # [必需] 软件版本号（如 6.4、8.0、latest）
icon: string             # [必需] 图标名称（如 globe, database, server 等）或相对文件路径
website: string          # [可选] 官方网站或源码地址
description: string      # [必需] 应用功能简要说明

# 反向代理与网关联动
gateway:
  is_web_app: boolean    # 是否为 Web 应用（默认 true）
  primary_port_key: string # 对应 ports 中声明的主 Web 端口 key（用于一键创建反向代理）
  default_subpath: string  # 默认访问子路径（默认为 "/"）

# 沙箱受控目录声明（安全约束）
# 部署时，CamoPanel 自动在 data/apps/<实例名>/ 下创建这些子目录
data_dirs:
  - string

# 端口映射定义
ports:
  - key: string          # [必需] 注入 compose 的环境变量名（如 HTTP_PORT、MYSQL_PORT）
    label: string        # [必需] 表单显示标签
    container_port: int  # [必需] 容器内目标端口（如 80、3306）
    protocol: string     # [可选] 协议，tcp（默认）或 udp
    default: int         # [必需] 默认宿主机端口
    description: string  # [可选] 端口用途提示说明
    required: boolean    # [可选] 是否必填，默认 true
    proxyable: boolean   # [可选] 是否可作为一键网站反代入口

# 用户配置参数定义
params:
  - key: string          # [必需] 环境变量名
    label: string        # [必需] 表单显示标签
    type: string         # [必需] 类型：string | password | number | boolean | select
    default: any         # [可选] 默认值。若为 password 且值为 "auto"，将自动生成 16 位强密码
    description: string  # [可选] 字段用途说明
    required: boolean    # [可选] 是否必填
    is_secret: boolean   # [可选] 是否敏感字段（界面掩码隐藏）
    auto_generate: boolean # [可选] 界面是否提供「随机生成密码」按钮
    options:             # [可选] 当 type 为 select 时的候选项列表
      - label: string
        value: string
    show_if:             # [可选] 字段级联依赖条件
      field: string      # 依赖的另一 param 变量名
      operator: string   # 比较操作符：equals | not_equals
      value: any         # 匹配的值
```

---

## 3. 编排模板 (`compose.yaml`) 规范

`compose.yaml` 严格遵循官方 Docker Compose Specification 标准。

### 3.1 变量注入规则
- 应用安装时，用户在界面中填写的端口映射与参数，将写入实例目录的 `.env` 文件。
- `compose.yaml` 中使用标准 `${VARIABLE_NAME}` 或 `${VARIABLE_NAME:-default}` 语法引用。
- CamoPanel 会自动预置以下系统变量：
  - `${APP_NAME}`：用户指定的实例名称；
  - `${APP_DIR}`：该实例在宿主机上的沙箱绝对根路径。

### 3.2 沙箱与卷挂载安全约束
> [!IMPORTANT]
> 为了杜绝容器逃逸与宿主机敏感文件损坏风险，CamoPanel 对 Compose 卷挂载实施强制防逃逸约束：
> 1. 挂载宿主机路径必须使用相对路径（如 `./data:/var/lib/mysql`）或命名卷（Named Volume）；
> 2. 严禁使用 `../` 跳出应用专属沙箱目录；
> 3. 严禁直接挂载宿主机绝对路径（如 `/etc`、`/var/run/docker.sock`、`/root`）。

---

## 4. 典型应用示例

### 4.1 WordPress (`apps/wordpress/`)

#### `apps/wordpress/app.yaml`:
```yaml
schema_version: "1.0"
id: "wordpress"
name: "WordPress & MySQL"
category: "web"
version: "6.4"
icon: "globe"
website: "https://wordpress.org"
description: "全球最流行的开源网站构建系统与内容管理平台，包含 MySQL 数据库与数据持久化约束。"

gateway:
  is_web_app: true
  primary_port_key: "WEB_PORT"
  default_subpath: "/"

data_dirs:
  - "data/wp"
  - "data/mysql"
  - "config"
  - "logs"

ports:
  - key: "WEB_PORT"
    label: "网站访问端口"
    container_port: 80
    protocol: "tcp"
    default: 8080
    description: "WordPress Web 服务宿主机访问端口"
    required: true
    proxyable: true

params:
  - key: "MYSQL_ROOT_PASSWORD"
    label: "MySQL Root 密码"
    type: "password"
    default: "auto"
    description: "数据库最高管理员密码"
    required: true
    is_secret: true
    auto_generate: true

  - key: "MYSQL_PASSWORD"
    label: "WordPress 数据库密码"
    type: "password"
    default: "auto"
    description: "WordPress 服务连接数据库密码"
    required: true
    is_secret: true
    auto_generate: true
```

#### `apps/wordpress/compose.yaml`:
```yaml
services:
  db:
    image: mysql:8.0
    container_name: ${APP_NAME}-db
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    volumes:
      - ./data/mysql:/var/lib/mysql

  wordpress:
    depends_on:
      - db
    image: wordpress:latest
    container_name: ${APP_NAME}-app
    restart: unless-stopped
    ports:
      - "${WEB_PORT}:80"
    environment:
      WORDPRESS_DB_HOST: db:3306
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: ${MYSQL_PASSWORD}
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - ./data/wp:/var/www/html
```
