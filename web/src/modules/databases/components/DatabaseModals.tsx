import { Alert, Button, Checkbox, Form, Input, Modal, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import type { DatabaseAccountItem } from "../types";

type DatabaseModalsProps = {
  quickCreateOpen: boolean;
  databaseModalOpen: boolean;
  accountModalOpen: boolean;
  grantModalOpen: boolean;
  passwordModalOpen: boolean;
  deleteDatabaseTarget: string | null;
  deleteAccountTarget: DatabaseAccountItem | null;
  passwordTarget: DatabaseAccountItem | null;
  submitting: boolean;
  deleteAccountEnabled: boolean;
  quickCreateForm: FormInstance;
  databaseForm: FormInstance;
  accountForm: FormInstance;
  grantForm: FormInstance;
  passwordForm: FormInstance;
  deleteDatabaseForm: FormInstance;
  databaseOptions: Array<{ label: string; value: string }>;
  accountOptions: Array<{ label: string; value: string }>;
  onCloseQuickCreate: () => void;
  onCloseDatabaseModal: () => void;
  onCloseAccountModal: () => void;
  onCloseGrantModal: () => void;
  onClosePasswordModal: () => void;
  onCloseDeleteDatabase: () => void;
  onCloseDeleteAccount: () => void;
  onCreateQuickWorkspace: (values: { name: string; password: string }) => Promise<void>;
  onCreateDatabase: (values: { name: string }) => Promise<void>;
  onCreateAccount: (values: { name: string; password: string; database_name?: string }) => Promise<void>;
  onGrantAccount: (values: { account_name: string; database_name: string }) => Promise<void>;
  onUpdateAccountPassword: (values: { password: string }) => Promise<void>;
  onDeleteDatabase: (values: { deleteAccount: boolean; accountName?: string }) => Promise<void>;
  onDeleteAccount: () => Promise<void>;
};

export function DatabaseModals({
  quickCreateOpen,
  databaseModalOpen,
  accountModalOpen,
  grantModalOpen,
  passwordModalOpen,
  deleteDatabaseTarget,
  deleteAccountTarget,
  passwordTarget,
  submitting,
  deleteAccountEnabled,
  quickCreateForm,
  databaseForm,
  accountForm,
  grantForm,
  passwordForm,
  deleteDatabaseForm,
  databaseOptions,
  accountOptions,
  onCloseQuickCreate,
  onCloseDatabaseModal,
  onCloseAccountModal,
  onCloseGrantModal,
  onClosePasswordModal,
  onCloseDeleteDatabase,
  onCloseDeleteAccount,
  onCreateQuickWorkspace,
  onCreateDatabase,
  onCreateAccount,
  onGrantAccount,
  onUpdateAccountPassword,
  onDeleteDatabase,
  onDeleteAccount,
}: DatabaseModalsProps) {
  return (
    <>
      <Modal
        open={quickCreateOpen}
        title="快速创建业务库"
        okText="立即创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseQuickCreate}
        onOk={() => void quickCreateForm.submit()}
        destroyOnClose
      >
        <Form form={quickCreateForm} layout="vertical" onFinish={onCreateQuickWorkspace}>
          <Alert
            showIcon
            type="info"
            message="会自动创建同名数据库和同名账号，并完成授权。"
            style={{ marginBottom: 16 }}
          />
          <Form.Item
            label="业务名"
            name="name"
            rules={[{ required: true, message: "请输入业务名" }]}
            extra="会同时用作数据库名和账号名。"
          >
            <Input placeholder="app_prod" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={databaseModalOpen}
        title="仅创建数据库"
        okText="创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseDatabaseModal}
        onOk={() => void databaseForm.submit()}
        destroyOnClose
      >
        <Form form={databaseForm} layout="vertical" onFinish={onCreateDatabase}>
          <Form.Item label="数据库名" name="name" rules={[{ required: true, message: "请输入数据库名" }]}>
            <Input placeholder="app_data" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={accountModalOpen}
        title="创建账号"
        okText="创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseAccountModal}
        onOk={() => void accountForm.submit()}
        destroyOnClose
      >
        <Form form={accountForm} layout="vertical" onFinish={onCreateAccount}>
          <Form.Item label="账号名" name="name" rules={[{ required: true, message: "请输入账号名" }]}>
            <Input placeholder="app_user" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Form.Item label="默认授权数据库" name="database_name">
            <Select allowClear showSearch options={databaseOptions} placeholder="可选，创建后直接完成授权" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={grantModalOpen}
        title="手动授权"
        okText="授权"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseGrantModal}
        onOk={() => void grantForm.submit()}
        destroyOnClose
      >
        <Form form={grantForm} layout="vertical" onFinish={onGrantAccount}>
          <Form.Item label="账号名" name="account_name" rules={[{ required: true, message: "请选择账号" }]}>
            <Select showSearch options={accountOptions} placeholder="选择账号" />
          </Form.Item>
          <Form.Item label="数据库名" name="database_name" rules={[{ required: true, message: "请选择数据库" }]}>
            <Select showSearch options={databaseOptions} placeholder="选择数据库" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={passwordModalOpen}
        title={passwordTarget ? `修改 ${passwordTarget.name} 密码` : "修改密码"}
        okText="更新"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onClosePasswordModal}
        onOk={() => void passwordForm.submit()}
        destroyOnClose
      >
        <Form form={passwordForm} layout="vertical" onFinish={onUpdateAccountPassword}>
          <Form.Item label="新密码" name="password" rules={[{ required: true, message: "请输入新密码" }]}>
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!deleteDatabaseTarget}
        title={deleteDatabaseTarget ? `删除数据库 ${deleteDatabaseTarget}` : "删除数据库"}
        okText="确认删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseDeleteDatabase}
        onOk={() => void deleteDatabaseForm.submit()}
        destroyOnClose
      >
        <Form form={deleteDatabaseForm} layout="vertical" onFinish={onDeleteDatabase}>
          <Alert
            showIcon
            type="warning"
            message="删除数据库后，库内数据会一并丢失。"
            style={{ marginBottom: 16 }}
          />
          <Form.Item name="deleteAccount" valuePropName="checked">
            <Checkbox>同步删除账号</Checkbox>
          </Form.Item>
          <Form.Item label="账号名" name="accountName">
            <Input disabled={!deleteAccountEnabled} placeholder="留空表示只删除数据库" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!deleteAccountTarget}
        title={deleteAccountTarget ? `删除账号 ${deleteAccountTarget.name}` : "删除账号"}
        okText="确认删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={onCloseDeleteAccount}
        onOk={() => void onDeleteAccount()}
        destroyOnClose
      >
        <Alert showIcon type="warning" message="删除账号后，将无法再使用该账号连接数据库。" />
      </Modal>
    </>
  );
}
