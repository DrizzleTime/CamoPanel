import {
  CheckCircleFilled,
  ClearOutlined,
  DeleteOutlined,
  InboxOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Progress,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from "antd";
import { useState } from "react";
import { bytesToSize } from "../../shared/lib/format";
import { executeCleanup, scanCleanup } from "./api";
import type { CleanupItem } from "./types";

const CATEGORY_META: Record<string, { icon: string; color: string }> = {
  apt_cache: { icon: "📦", color: "#1677ff" },
  yum_cache: { icon: "📦", color: "#1677ff" },
  journal_logs: { icon: "📋", color: "#fa8c16" },
  tmp_files: { icon: "🗂️", color: "#722ed1" },
  old_logs: { icon: "📄", color: "#13c2c2" },
  user_cache: { icon: "💾", color: "#52c41a" },
};

type Phase = "idle" | "scanning" | "scanned" | "cleaning" | "done";

export function CleanupPage() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [items, setItems] = useState<Record<string, CleanupItem>>({});
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string>();
  const [cleanedCount, setCleanedCount] = useState(0);

  const totalSize = Object.values(items).reduce((sum, item) => sum + item.size, 0);
  const selectedSize = Array.from(selected).reduce((sum, key) => sum + (items[key]?.size ?? 0), 0);

  const handleScan = async () => {
    setPhase("scanning");
    setError(undefined);
    setSelected(new Set());
    setCleanedCount(0);
    try {
      const result = await scanCleanup();
      setItems(result.items);
      const nonEmpty = Object.entries(result.items)
        .filter(([, v]) => v.size > 0)
        .map(([k]) => k);
      setSelected(new Set(nonEmpty));
      setPhase("scanned");
    } catch (err) {
      setError(err instanceof Error ? err.message : "扫描失败");
      setPhase("idle");
    }
  };

  const handleClean = async () => {
    if (selected.size === 0) {
      message.warning("请先选择需要清理的项目");
      return;
    }
    setPhase("cleaning");
    try {
      const result = await executeCleanup(Array.from(selected));
      setCleanedCount(result.cleaned);
      setPhase("done");
      message.success(`已完成 ${result.cleaned} 项清理`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "清理失败");
      setPhase("scanned");
    }
  };

  const toggleItem = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleAll = () => {
    const nonEmpty = Object.entries(items)
      .filter(([, v]) => v.size > 0)
      .map(([k]) => k);
    if (selected.size === nonEmpty.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(nonEmpty));
    }
  };

  return (
    <div className="page-grid">
      <div className="page-inline-bar">
        <Typography.Text type="secondary">
          扫描并清理系统中的缓存、日志和临时文件，释放磁盘空间。
        </Typography.Text>
      </div>

      {error ? <Alert showIcon type="error" message={error} closable onClose={() => setError(undefined)} /> : null}

      {phase === "idle" ? (
        <Card className="glass-card cleanup-hero-card" style={{ borderRadius: 18 }}>
          <div className="cleanup-hero">
            <div className="cleanup-hero-icon">
              <SearchOutlined style={{ fontSize: 48, color: "#1677ff" }} />
            </div>
            <Typography.Title level={3} style={{ margin: 0 }}>
              系统垃圾清理
            </Typography.Title>
            <Typography.Paragraph type="secondary" style={{ maxWidth: 480, textAlign: "center", margin: 0 }}>
              点击扫描按钮检查系统中的缓存文件、过期日志和临时数据，选择性清理以释放磁盘空间。
            </Typography.Paragraph>
            <Button
              type="primary"
              size="large"
              icon={<SearchOutlined />}
              onClick={() => void handleScan()}
              style={{ marginTop: 8 }}
            >
              开始扫描
            </Button>
          </div>
        </Card>
      ) : null}

      {phase === "scanning" ? (
        <Card className="glass-card cleanup-hero-card" style={{ borderRadius: 18 }}>
          <div className="cleanup-hero">
            <Spin size="large" />
            <Typography.Title level={4} style={{ margin: 0 }}>
              正在扫描系统...
            </Typography.Title>
            <Typography.Text type="secondary">请稍候，正在检查各项缓存和临时文件</Typography.Text>
          </div>
        </Card>
      ) : null}

      {phase === "scanned" || phase === "cleaning" ? (
        <>
          <Card className="glass-card" style={{ borderRadius: 18 }}>
            <div className="cleanup-summary-bar">
              <div className="cleanup-summary-left">
                <Typography.Title level={4} style={{ margin: 0 }}>
                  扫描结果
                </Typography.Title>
                <Space size={16}>
                  <Typography.Text type="secondary">
                    共发现 <Typography.Text strong>{bytesToSize(totalSize)}</Typography.Text> 可清理空间
                  </Typography.Text>
                  {selected.size > 0 ? (
                    <Tag color="blue">
                      已选 {selected.size} 项 · {bytesToSize(selectedSize)}
                    </Tag>
                  ) : null}
                </Space>
              </div>
              <Space>
                <Button icon={<ReloadOutlined />} onClick={() => void handleScan()}>
                  重新扫描
                </Button>
                <Button
                  type="primary"
                  danger
                  icon={<ClearOutlined />}
                  loading={phase === "cleaning"}
                  disabled={selected.size === 0}
                  onClick={() => void handleClean()}
                >
                  清理选中项
                </Button>
              </Space>
            </div>
          </Card>

          <div className="cleanup-select-bar">
            <Checkbox
              checked={
                selected.size > 0 &&
                selected.size ===
                  Object.entries(items).filter(([, v]) => v.size > 0).length
              }
              indeterminate={
                selected.size > 0 &&
                selected.size <
                  Object.entries(items).filter(([, v]) => v.size > 0).length
              }
              onChange={toggleAll}
            >
              全选
            </Checkbox>
          </div>

          <div className="cleanup-item-grid">
            {Object.entries(items).map(([key, item]) => {
              const meta = CATEGORY_META[key] ?? { icon: "📁", color: "#595959" };
              const isEmpty = item.size === 0;
              const isSelected = selected.has(key);
              return (
                <Card
                  key={key}
                  className={`glass-card cleanup-item-card ${isSelected && !isEmpty ? "cleanup-item-selected" : ""}`}
                  style={{
                    borderRadius: 16,
                    opacity: isEmpty ? 0.5 : 1,
                    cursor: isEmpty ? "default" : "pointer",
                  }}
                  onClick={() => !isEmpty && toggleItem(key)}
                >
                  <div className="cleanup-item-body">
                    <div className="cleanup-item-top">
                      <span className="cleanup-item-emoji">{meta.icon}</span>
                      {!isEmpty ? (
                        <Checkbox
                          checked={isSelected}
                          onClick={(e) => e.stopPropagation()}
                          onChange={() => toggleItem(key)}
                        />
                      ) : null}
                    </div>
                    <div className="cleanup-item-info">
                      <Typography.Text strong>{item.description}</Typography.Text>
                      <Typography.Title
                        level={4}
                        style={{ margin: 0, color: isEmpty ? "#bfbfbf" : meta.color }}
                      >
                        {bytesToSize(item.size)}
                      </Typography.Title>
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>
        </>
      ) : null}

      {phase === "done" ? (
        <Card className="glass-card cleanup-hero-card" style={{ borderRadius: 18 }}>
          <div className="cleanup-hero">
            <CheckCircleFilled style={{ fontSize: 56, color: "#52c41a" }} />
            <Typography.Title level={3} style={{ margin: 0 }}>
              清理完成
            </Typography.Title>
            <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
              已成功完成 {cleanedCount} 项清理任务
            </Typography.Paragraph>
            <Space style={{ marginTop: 8 }}>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={() => void handleScan()}
              >
                重新扫描
              </Button>
              <Button onClick={() => setPhase("idle")}>返回</Button>
            </Space>
          </div>
        </Card>
      ) : null}
    </div>
  );
}
