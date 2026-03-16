import {
  ArrowUpOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  FileOutlined,
  FileTextOutlined,
  FolderAddOutlined,
  FolderOpenOutlined,
  ReloadOutlined,
  SwapOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { ApiError, apiRequest, bytesToSize } from "../lib/api";
import type { FileEntry, FileListResponse, FileReadResponse } from "../lib/types";

type CreateKind = "file" | "directory";

type CreateFormValues = {
  name: string;
  content?: string;
};

type MoveFormValues = {
  to_path: string;
};

export function FilesPage() {
  const uploadInputRef = useRef<HTMLInputElement | null>(null);
  const [items, setItems] = useState<FileEntry[]>([]);
  const [currentPath, setCurrentPath] = useState("/");
  const [parentPath, setParentPath] = useState("");
  const [pathInput, setPathInput] = useState("/");
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createKind, setCreateKind] = useState<CreateKind>("file");
  const [moveTarget, setMoveTarget] = useState<FileEntry | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorLoading, setEditorLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [activeFile, setActiveFile] = useState<FileReadResponse | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [createForm] = Form.useForm<CreateFormValues>();
  const [moveForm] = Form.useForm<MoveFormValues>();

  const loadFiles = async (targetPath = currentPath) => {
    setLoading(true);
    try {
      const response = await apiRequest<FileListResponse>(
        `/api/files/list?path=${encodeURIComponent(targetPath)}`,
      );
      setItems(response.items);
      setCurrentPath(response.current_path);
      setPathInput(response.current_path);
      setParentPath(response.parent_path);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadFiles("/");
  }, []);

  const openFile = async (item: FileEntry) => {
    setEditorOpen(true);
    setEditorLoading(true);
    try {
      const response = await apiRequest<FileReadResponse>(
        `/api/files/read?path=${encodeURIComponent(item.path)}`,
      );
      setActiveFile(response);
      setEditorContent(response.content);
    } catch (error) {
      setEditorOpen(false);
      setActiveFile(null);
      message.error(getErrorMessage(error));
    } finally {
      setEditorLoading(false);
    }
  };

  const handleEnter = (item: FileEntry) => {
    if (item.type === "directory") {
      void loadFiles(item.path);
      return;
    }
    void openFile(item);
  };

  const handleSave = async () => {
    if (!activeFile || activeFile.is_binary) return;
    setSaving(true);
    try {
      await apiRequest<{ path: string }>("/api/files/write", {
        method: "POST",
        body: JSON.stringify({
          path: activeFile.path,
          content: editorContent,
        }),
      });
      setActiveFile({ ...activeFile, content: editorContent });
      message.success("文件已保存");
      void loadFiles(currentPath);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const handleOpenCreate = (kind: CreateKind) => {
    setCreateKind(kind);
    createForm.resetFields();
    setCreateOpen(true);
  };

  const handleCreate = async (values: CreateFormValues) => {
    const targetPath = buildChildPath(currentPath, values.name);
    setSubmitting(true);
    try {
      if (createKind === "directory") {
        await apiRequest<{ path: string }>("/api/files/mkdir", {
          method: "POST",
          body: JSON.stringify({ path: targetPath }),
        });
      } else {
        await apiRequest<{ path: string }>("/api/files/create", {
          method: "POST",
          body: JSON.stringify({ path: targetPath, content: values.content ?? "" }),
        });
      }
      message.success(createKind === "directory" ? "目录已创建" : "文件已创建");
      setCreateOpen(false);
      createForm.resetFields();
      void loadFiles(currentPath);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const openMoveModal = (item: FileEntry) => {
    setMoveTarget(item);
    moveForm.setFieldsValue({ to_path: item.path });
  };

  const handleMove = async (values: MoveFormValues) => {
    if (!moveTarget) return;
    setSubmitting(true);
    try {
      await apiRequest<{ path: string }>("/api/files/move", {
        method: "POST",
        body: JSON.stringify({
          from_path: moveTarget.path,
          to_path: values.to_path,
        }),
      });
      if (activeFile?.path === moveTarget.path) {
        setActiveFile({
          ...activeFile,
          path: values.to_path,
          name: values.to_path.split("/").filter(Boolean).pop() || activeFile.name,
        });
      }
      message.success("移动完成");
      setMoveTarget(null);
      moveForm.resetFields();
      void loadFiles(currentPath);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = (item: FileEntry) => {
    Modal.confirm({
      title: `确认删除 ${item.name}？`,
      content: "删除会直接生效，目录会递归删除。",
      okText: "删除",
      cancelText: "取消",
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await apiRequest<{ path: string }>("/api/files/delete", {
            method: "POST",
            body: JSON.stringify({ path: item.path }),
          });
          if (activeFile?.path === item.path) {
            setEditorOpen(false);
            setActiveFile(null);
            setEditorContent("");
          }
          message.success("删除完成");
          await loadFiles(currentPath);
        } catch (error) {
          message.error(getErrorMessage(error));
        }
      },
    });
  };

  const handleUploadClick = () => {
    uploadInputRef.current?.click();
  };

  const handleUploadChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    if (!files.length) return;

    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("path", currentPath);
      files.forEach((file) => formData.append("files", file));
      await apiRequest<{ items: string[] }>("/api/files/upload", {
        method: "POST",
        body: formData,
      });
      message.success(`已上传 ${files.length} 个文件`);
      await loadFiles(currentPath);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      event.target.value = "";
      setUploading(false);
    }
  };

  const handleDownload = (path: string) => {
    const link = document.createElement("a");
    link.href = `/api/files/download?path=${encodeURIComponent(path)}`;
    document.body.appendChild(link);
    link.click();
    link.remove();
  };

  return (
    <div className="page-grid">
      <Card className="glass-card">
        <div className="files-toolbar">
          <Input.Search
            value={pathInput}
            placeholder="/etc/nginx"
            enterButton="打开路径"
            onChange={(event) => setPathInput(event.target.value)}
            onSearch={(value) => void loadFiles(value)}
          />
          <Space wrap>
            <Button
              icon={<ArrowUpOutlined />}
              onClick={() => void loadFiles(parentPath)}
              disabled={!parentPath}
            >
              上级
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadFiles(currentPath)}>
              刷新
            </Button>
            <Button icon={<UploadOutlined />} loading={uploading} onClick={handleUploadClick}>
              上传
            </Button>
            <Button icon={<FileTextOutlined />} onClick={() => handleOpenCreate("file")}>
              新建文件
            </Button>
            <Button icon={<FolderAddOutlined />} onClick={() => handleOpenCreate("directory")}>
              新建目录
            </Button>
          </Space>
        </div>

        <div className="files-current-path">
          <Typography.Text type="secondary">当前路径</Typography.Text>
          <Typography.Text code>{currentPath}</Typography.Text>
        </div>

        <input
          ref={uploadInputRef}
          type="file"
          multiple
          hidden
          onChange={(event) => void handleUploadChange(event)}
        />
      </Card>

      <Card className="glass-card">
        <Table
          rowKey="path"
          loading={loading}
          dataSource={items}
          pagination={false}
          locale={{ emptyText: <Empty description="当前目录为空" /> }}
          onRow={(record) => ({
            onClick: () => handleEnter(record),
          })}
          columns={[
            {
              title: "名称",
              dataIndex: "name",
              render: (_, record: FileEntry) => (
                <Space>
                  {record.type === "directory" ? <FolderOpenOutlined /> : <FileOutlined />}
                  <Typography.Text>{record.name}</Typography.Text>
                </Space>
              ),
            },
            {
              title: "类型",
              dataIndex: "type",
              width: 120,
              render: (value: FileEntry["type"]) => (
                <Tag color={value === "directory" ? "blue" : value === "symlink" ? "gold" : "default"}>
                  {value === "directory" ? "目录" : value === "symlink" ? "链接" : "文件"}
                </Tag>
              ),
            },
            {
              title: "大小",
              dataIndex: "size",
              width: 140,
              render: (value: number, record: FileEntry) =>
                record.type === "directory" ? "-" : bytesToSize(value),
            },
            {
              title: "权限",
              dataIndex: "mode",
              width: 140,
            },
            {
              title: "修改时间",
              dataIndex: "modified_at",
              width: 220,
              render: (value: string) => formatDate(value),
            },
            {
              title: "动作",
              width: 280,
              render: (_, record: FileEntry) => (
                <Space onClick={(event) => event.stopPropagation()}>
                  <Button size="small" onClick={() => handleEnter(record)}>
                    {record.type === "directory" ? "进入" : "打开"}
                  </Button>
                  {record.type !== "directory" ? (
                    <Button
                      size="small"
                      icon={<DownloadOutlined />}
                      onClick={() => handleDownload(record.path)}
                    />
                  ) : null}
                  <Button
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => openMoveModal(record)}
                  />
                  <Button
                    danger
                    size="small"
                    icon={<DeleteOutlined />}
                    onClick={() => handleDelete(record)}
                  />
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Drawer
        open={editorOpen}
        width={820}
        title={activeFile?.name || "文件"}
        onClose={() => {
          setEditorOpen(false);
          setActiveFile(null);
          setEditorContent("");
        }}
        extra={
          <Space>
            {activeFile ? (
              <Button icon={<DownloadOutlined />} onClick={() => handleDownload(activeFile.path)}>
                下载
              </Button>
            ) : null}
            <Button
              type="primary"
              loading={saving}
              disabled={!activeFile || activeFile.is_binary}
              onClick={() => void handleSave()}
            >
              保存
            </Button>
          </Space>
        }
      >
        {editorLoading ? (
          <Typography.Text type="secondary">正在读取文件...</Typography.Text>
        ) : activeFile ? (
          <Space direction="vertical" size="large" style={{ width: "100%" }}>
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="路径">{activeFile.path}</Descriptions.Item>
              <Descriptions.Item label="大小">{bytesToSize(activeFile.size)}</Descriptions.Item>
              <Descriptions.Item label="权限">{activeFile.mode}</Descriptions.Item>
              <Descriptions.Item label="修改时间">
                {formatDate(activeFile.modified_at)}
              </Descriptions.Item>
            </Descriptions>

            {activeFile.is_binary ? (
              <Alert
                showIcon
                type="info"
                message="这是一个二进制文件，当前只支持下载，不支持在线编辑。"
              />
            ) : (
              <Input.TextArea
                value={editorContent}
                onChange={(event) => setEditorContent(event.target.value)}
                autoSize={{ minRows: 20, maxRows: 30 }}
                className="file-editor-area"
              />
            )}
          </Space>
        ) : null}
      </Drawer>

      <Modal
        open={createOpen}
        title={createKind === "directory" ? "新建目录" : "新建文件"}
        okText="创建"
        cancelText="取消"
        confirmLoading={submitting}
        destroyOnClose
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={() => void createForm.submit()}
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            label={createKind === "directory" ? "目录名" : "文件名"}
            name="name"
            rules={[{ required: true, whitespace: true, message: "请输入名称" }]}
            extra="只填写当前目录下的名称，不要填写完整绝对路径。"
          >
            <Input placeholder={createKind === "directory" ? "logs" : "config.json"} />
          </Form.Item>
          {createKind === "file" ? (
            <Form.Item label="初始内容" name="content">
              <Input.TextArea autoSize={{ minRows: 6, maxRows: 12 }} />
            </Form.Item>
          ) : null}
        </Form>
      </Modal>

      <Modal
        open={!!moveTarget}
        title="移动 / 重命名"
        okText="确认"
        cancelText="取消"
        confirmLoading={submitting}
        destroyOnClose
        onCancel={() => {
          setMoveTarget(null);
          moveForm.resetFields();
        }}
        onOk={() => void moveForm.submit()}
      >
        <Form form={moveForm} layout="vertical" onFinish={handleMove}>
          <Form.Item label="源路径">
            <Input value={moveTarget?.path} disabled />
          </Form.Item>
          <Form.Item
            label="目标路径"
            name="to_path"
            rules={[{ required: true, message: "请输入目标绝对路径" }]}
          >
            <Input prefix={<SwapOutlined />} placeholder="/etc/nginx/nginx.conf" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function buildChildPath(basePath: string, name: string) {
  const cleanName = name.replace(/^\/+/, "").trim();
  if (!cleanName) {
    return basePath;
  }
  if (basePath === "/") {
    return `/${cleanName}`;
  }
  return `${basePath.replace(/\/+$/, "")}/${cleanName}`;
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}

function getErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "请求失败";
}
