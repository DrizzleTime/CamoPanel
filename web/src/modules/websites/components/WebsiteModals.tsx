import { Alert, Button, Form, Input, InputNumber, Modal, Radio, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import type { Project, TemplateSpec } from "../../../shared/types";
import type {
  CertificateFormValues,
  EnvironmentFormValues,
  OpenRestyStatus,
  Website,
  WebsiteFormValues,
} from "../types";

type WebsiteModalsProps = {
  status: OpenRestyStatus | null;
  editingWebsite: Website | null;
  modalOpen: boolean;
  previewOpen: boolean;
  previewTitle: string;
  previewContent: string;
  environmentModalOpen: boolean;
  certificateModalOpen: boolean;
  submitting: boolean;
  environmentSubmitting: boolean;
  certificateSubmitting: boolean;
  websiteType: WebsiteFormValues["type"];
  rewriteMode: WebsiteFormValues["rewrite_mode"];
  phpProjects: Project[];
  phpTemplate: TemplateSpec | null;
  form: FormInstance<WebsiteFormValues>;
  environmentForm: FormInstance<EnvironmentFormValues>;
  certificateForm: FormInstance<CertificateFormValues>;
  certificateReady: boolean;
  onCloseWebsiteModal: () => void;
  onClosePreview: () => void;
  onCloseEnvironmentModal: () => void;
  onCloseCertificateModal: () => void;
  onSubmitWebsite: (values: WebsiteFormValues) => Promise<void>;
  onSubmitEnvironment: (values: EnvironmentFormValues) => Promise<void>;
  onSubmitCertificate: (values: CertificateFormValues) => Promise<void>;
  onHandleWebsiteTypeChange: (nextType: WebsiteFormValues["type"]) => void;
  projectConfigText: (project: Project, key: string) => string;
  projectConfigNumber: (project: Project, key: string) => number;
  rewritePresetOptions: Array<{ value: string; label: string }>;
};

export function WebsiteModals({
  status,
  editingWebsite,
  modalOpen,
  previewOpen,
  previewTitle,
  previewContent,
  environmentModalOpen,
  certificateModalOpen,
  submitting,
  environmentSubmitting,
  certificateSubmitting,
  websiteType,
  rewriteMode,
  phpProjects,
  phpTemplate,
  form,
  environmentForm,
  certificateForm,
  certificateReady,
  onCloseWebsiteModal,
  onClosePreview,
  onCloseEnvironmentModal,
  onCloseCertificateModal,
  onSubmitWebsite,
  onSubmitEnvironment,
  onSubmitCertificate,
  onHandleWebsiteTypeChange,
  projectConfigText,
  projectConfigNumber,
  rewritePresetOptions,
}: WebsiteModalsProps) {
  return (
    <>
      <Modal
        open={modalOpen}
        title={editingWebsite ? `配置站点 ${editingWebsite.name}` : "创建站点"}
        width={760}
        okText={editingWebsite ? "保存配置" : "立即创建"}
        cancelText="取消"
        onCancel={onCloseWebsiteModal}
        onOk={() => void form.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            type: "static",
            domains: "",
            rewrite_mode: "off",
            index_files: "index.html index.htm",
          }}
          onFinish={onSubmitWebsite}
        >
          {!status?.ready ? (
            <Alert
              showIcon
              type="warning"
              message={status?.message || "OpenResty 当前不可用"}
              style={{ marginBottom: 16 }}
            />
          ) : null}

          <Form.Item
            label="站点名"
            name="name"
            rules={[{ required: true, message: "请输入站点名" }]}
            extra="只允许小写字母、数字、下划线和中划线。当前版本不支持修改站点名。"
          >
            <Input placeholder="my-site" disabled={!!editingWebsite} />
          </Form.Item>

          <Form.Item
            label="站点模式"
            name="type"
            rules={[{ required: true, message: "请选择站点模式" }]}
          >
            <Radio.Group
              onChange={(event) => onHandleWebsiteTypeChange(event.target.value as WebsiteFormValues["type"])}
              options={[
                { label: "静态站点", value: "static" },
                { label: "PHP 站点", value: "php" },
                { label: "整站反向代理", value: "proxy" },
              ]}
            />
          </Form.Item>

          <Form.Item
            label="主域名"
            name="domain"
            rules={[{ required: true, message: "请输入主域名" }]}
          >
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item
            label="附加域名"
            name="domains"
            extra="多个域名用逗号、空格或换行分隔。"
          >
            <Input.TextArea rows={3} placeholder="www.example.com, static.example.com" />
          </Form.Item>

          {websiteType === "static" || websiteType === "php" ? (
            <>
              {websiteType === "php" && phpProjects.length === 0 ? (
                <Alert
                  showIcon
                  type="warning"
                  message="还没有可用的 PHP 环境"
                  description="请先在上方“环境”页创建 php-fpm 实例，再回来绑定站点。"
                  style={{ marginBottom: 16 }}
                />
              ) : null}

              <Form.Item
                label="站点目录"
                name="root_path"
                extra="支持相对路径或绝对路径，但必须位于 OpenResty 站点挂载目录下。留空时默认使用站点名目录。"
              >
                <Input placeholder="test 或 /root/CamoPanel/server/data/openresty/www/test" />
              </Form.Item>

              <Form.Item
                label="首页文件"
                name="index_files"
                extra="使用空格分隔多个首页文件。"
              >
                <Input placeholder={websiteType === "php" ? "index.php index.html index.htm" : "index.html index.htm"} />
              </Form.Item>

              {websiteType === "php" ? (
                <Form.Item
                  label="PHP 环境"
                  name="php_project_id"
                  rules={[{ required: true, message: "请选择 PHP 环境" }]}
                  extra="站点会把 .php 请求转发到选中的 php-fpm 实例。"
                >
                  <Select
                    options={phpProjects.map((item) => ({
                      value: item.id,
                      label: `${item.name} / PHP ${projectConfigText(item, "php_version") || "-"} / 127.0.0.1:${projectConfigNumber(item, "port") || "-"}`,
                    }))}
                    placeholder="选择 PHP 环境"
                  />
                </Form.Item>
              ) : null}

              <Form.Item label="伪静态" name="rewrite_mode">
                <Radio.Group
                  options={[
                    { label: "关闭", value: "off" },
                    { label: "预设", value: "preset" },
                    { label: "自定义", value: "custom" },
                  ]}
                />
              </Form.Item>

              {rewriteMode === "preset" ? (
                <Form.Item
                  label="伪静态预设"
                  name="rewrite_preset"
                  rules={[{ required: true, message: "请选择伪静态预设" }]}
                >
                  <Select options={rewritePresetOptions} placeholder="选择伪静态预设" />
                </Form.Item>
              ) : null}

              {rewriteMode === "custom" ? (
                <Form.Item
                  label="自定义伪静态规则"
                  name="rewrite_rules"
                  rules={[{ required: true, message: "请输入自定义规则" }]}
                  extra="直接填写 location / 内部规则片段。"
                >
                  <Input.TextArea
                    rows={8}
                    placeholder={
                      websiteType === "php"
                        ? "try_files $uri $uri/ /index.php?$query_string;"
                        : "try_files $uri $uri/ /index.html;"
                    }
                  />
                </Form.Item>
              ) : null}
            </>
          ) : (
            <Form.Item
              label="代理地址"
              name="proxy_pass"
              rules={[{ required: true, message: "请输入代理地址" }]}
              extra="示例：http://127.0.0.1:3000"
            >
              <Input placeholder="http://127.0.0.1:3000" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        open={previewOpen}
        title={previewTitle}
        footer={[
          <Button key="close" onClick={onClosePreview}>
            关闭
          </Button>,
        ]}
        width={860}
        onCancel={onClosePreview}
      >
        <pre className="mono-box">{previewContent}</pre>
      </Modal>

      <Modal
        open={environmentModalOpen}
        title="新建 PHP 环境"
        okText="创建环境"
        cancelText="取消"
        onCancel={onCloseEnvironmentModal}
        onOk={() => void environmentForm.submit()}
        confirmLoading={environmentSubmitting}
        destroyOnClose
      >
        <Form
          form={environmentForm}
          layout="vertical"
          initialValues={{ name: "", php_version: "8.3", port: 9000 }}
          onFinish={onSubmitEnvironment}
        >
          <Form.Item
            label="环境名"
            name="name"
            rules={[{ required: true, message: "请输入环境名" }]}
            extra="只允许小写字母、数字、下划线和中划线。"
          >
            <Input placeholder="php83-blog" />
          </Form.Item>

          <Form.Item
            label="PHP 版本"
            name="php_version"
            rules={[{ required: true, message: "请选择 PHP 版本" }]}
          >
            <Select
              options={[
                { value: "8.1", label: "PHP 8.1" },
                { value: "8.2", label: "PHP 8.2" },
                { value: "8.3", label: "PHP 8.3" },
              ]}
            />
          </Form.Item>

          <Form.Item
            label="FPM 端口"
            name="port"
            rules={[{ required: true, message: "请输入 FPM 端口" }]}
            extra="会绑定到宿主机 127.0.0.1，仅供 OpenResty 转发。"
          >
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={certificateModalOpen}
        title="申请证书"
        okText="立即申请"
        cancelText="取消"
        onCancel={onCloseCertificateModal}
        onOk={() => void certificateForm.submit()}
        confirmLoading={certificateSubmitting}
        destroyOnClose
      >
        <Form
          form={certificateForm}
          layout="vertical"
          initialValues={{ domain: "", email: "" }}
          onFinish={onSubmitCertificate}
        >
          {!certificateReady ? (
            <Alert
              showIcon
              type="warning"
              message={status?.message || "OpenResty 当前不可用"}
              style={{ marginBottom: 16 }}
            />
          ) : null}

          <Form.Item
            label="域名"
            name="domain"
            rules={[{ required: true, message: "请输入域名" }]}
            extra="第一版只支持单域名证书。"
          >
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            name="email"
            rules={[{ required: true, message: "请输入邮箱" }, { type: "email", message: "邮箱格式不正确" }]}
            extra="Let's Encrypt 注册和证书通知会使用这个邮箱。"
          >
            <Input placeholder="admin@example.com" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
