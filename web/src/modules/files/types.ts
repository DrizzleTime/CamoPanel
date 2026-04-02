export type FileEntry = {
  name: string;
  path: string;
  type: "file" | "directory" | "symlink";
  size: number;
  mode: string;
  modified_at: string;
};

export type FileListResponse = {
  current_path: string;
  parent_path: string;
  items: FileEntry[];
};

export type FileReadResponse = {
  path: string;
  name: string;
  size: number;
  mode: string;
  modified_at: string;
  content: string;
  is_binary: boolean;
};
