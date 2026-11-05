// Package version 提供仓库级发布版本号，作为三端共用的唯一来源。
package version

// Version 是发布版本号；发版只改这一行，并在变更合并到 main 后打同名 tag（docs/002 §1.1）。
const Version = "0.1.0"
