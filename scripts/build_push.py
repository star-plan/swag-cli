#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Go项目构建与远程部署脚本

功能：
1. 获取最新git tag作为版本号
2. 编译Go项目为指定平台的可执行文件
3. 如果启用部署，将可执行文件推送到服务器并安装到指定目录

配置项(环境变量或默认值)：
- TARGET_OS: 目标操作系统，默认: linux
- TARGET_ARCH: 目标架构，默认: amd64
- EXECUTABLE_NAME: 可执行文件名称，默认: swag-cli
- REMOTE_HOST: 远程服务器配置，如: user@server-ip 或 ssh config host别名
- REMOTE_TEMP_PATH: 远程临时目录，默认: /tmp
- REMOTE_INSTALL_PATH: 远程安装目录，默认: /usr/local/bin
- ENABLED_DEPLOY: 是否启用部署，默认: True
- USE_SSH_TTY: SSH是否分配伪终端(用于sudo需要密码的情况)，默认: True
- SUDO_NOPASSWD: 服务器sudo是否配置了NOPASSWD，默认: False

注意事项:
1. 如果服务器sudo需要密码:
   - 方案A(推荐): 配置sudo NOPASSWD，设置 SUDO_NOPASSWD=True
     在服务器上运行: sudo visudo
     添加: your_user ALL=(ALL) NOPASSWD: /bin/mv, /bin/chmod
   - 方案B: 使用SSH TTY，设置 USE_SSH_TTY=True (默认)
     这会在SSH命令中添加 -t 参数，允许交互式输入密码
2. 如果服务器sudo已配置NOPASSWD，设置 SUDO_NOPASSWD=True 可以避免TTY相关问题
"""

import os
import sys
import subprocess
import threading
from typing import Optional, Tuple, Dict

# 默认配置
DEFAULTS = {
    "TARGET_OS": "linux",
    "TARGET_ARCH": "amd64",
    "EXECUTABLE_NAME": "swag-cli",
    'REMOTE_HOST': 'deali.sg',
    'REMOTE_TEMP_PATH': '/tmp',
    'REMOTE_INSTALL_PATH': '/usr/local/bin',
    "ENABLED_DEPLOY": True,
    "USE_SSH_TTY": True,  # 使用SSH TTY以支持sudo密码输入
    "SUDO_NOPASSWD": False,  # 服务器sudo是否配置了NOPASSWD
}


class ProgressDisplay:
    """
    管理一个持久的状态行，同时允许其他输出滚动显示。
    """

    def __init__(self):
        self.status_line = ""
        self.lock = threading.Lock()

    def set_status(self, status: str):
        """设置或更新状态行文本"""
        with self.lock:
            sys.stdout.write('\r\033[K')  # 清空当前行
            self.status_line = status
            sys.stdout.write(self.status_line)
            sys.stdout.flush()

    def print_output(self, line: str):
        """在状态行下方打印一行输出"""
        with self.lock:
            # 清空当前行（即状态行）
            sys.stdout.write('\r\033[K')
            # 打印实际的命令输出行
            sys.stdout.write(line)
            # 重新绘制状态行
            sys.stdout.write(self.status_line)
            sys.stdout.flush()

    def finish_step(self, final_status: str):
        """完成一个步骤，将最终状态打印为普通行"""
        with self.lock:
            # 清空状态行
            sys.stdout.write('\r\033[K')
            # 打印最终状态
            sys.stdout.write(final_status + '\n')
            sys.stdout.flush()
            self.status_line = ""


def get_config(key: str) -> str | object:
    """获取配置值，优先使用环境变量，否则使用默认值"""
    val = os.environ.get(key)
    if val is not None:
        # 如果是布尔值配置，尝试转换
        if key in ("ENABLED_DEPLOY", "USE_SSH_TTY", "SUDO_NOPASSWD"):
            return val.lower() in ('true', '1', 'yes', 'on')
        return val
    return DEFAULTS.get(key, '')


def _reader_thread(pipe, lines_list, progress_display: Optional[ProgressDisplay]):
    """在独立线程中读取管道输出"""
    try:
        for line in iter(pipe.readline, ''):
            lines_list.append(line)
            if progress_display:
                progress_display.print_output(line)
    except Exception as e:
        error_msg = f"读取输出错误: {e}\n"
        lines_list.append(error_msg)
        if progress_display:
            progress_display.print_output(error_msg)
    finally:
        pipe.close()


def run_cmd(cmd: str, progress_display: Optional[ProgressDisplay] = None, env: Optional[Dict[str, str]] = None, check: bool = True) -> Tuple[int, str, str]:
    """
    执行命令并实时显示输出，同时捕获输出内容。
    返回状态码、stdout和stderr。
    check: 如果为True，当命令返回非0状态码时退出脚本
    """
    # 如果没有传入 env，使用当前进程的环境变量
    if env is None:
        run_env = os.environ.copy()
    else:
        run_env = env

    if progress_display:
        progress_display.print_output(f"执行命令: {cmd}\n")
    else:
        print(f"执行命令: {cmd}")

    process = subprocess.Popen(
        cmd,
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
        universal_newlines=True,
        encoding='utf-8',
        errors='replace',
        env=run_env
    )

    stdout_lines = []
    stderr_lines = []

    stdout_thread = threading.Thread(
        target=_reader_thread,
        args=(process.stdout, stdout_lines, progress_display)
    )
    stderr_thread = threading.Thread(
        target=_reader_thread,
        args=(process.stderr, stderr_lines, progress_display)
    )

    stdout_thread.start()
    stderr_thread.start()

    stdout_thread.join()
    stderr_thread.join()

    returncode = process.wait()

    stdout = ''.join(stdout_lines)
    stderr = ''.join(stderr_lines)

    if returncode != 0 and check:
        if progress_display:
            progress_display.print_output(f"\n❌ 命令执行失败 (返回码: {returncode})\n")
        else:
            print(f"\n错误: 命令 '{cmd}' 执行失败 (返回码: {returncode})")
            print(stderr)
        sys.exit(1)

    return returncode, stdout, stderr


def get_latest_tag() -> str:
    """获取最新git tag"""
    try:
        returncode, tag, _ = run_cmd("git describe --tags --abbrev=0", check=False)
        if returncode != 0:
            return "dev"
        tag = tag.strip()
        return tag if tag else "dev"
    except Exception:
        # 如果获取失败，返回 dev
        return "dev"


def build_project(version: str, progress: ProgressDisplay) -> str:
    """编译Go项目"""
    os_name = str(get_config('TARGET_OS'))
    arch = str(get_config('TARGET_ARCH'))
    exe_name = str(get_config('EXECUTABLE_NAME'))

    # 创建 build 目录
    if not os.path.exists('build'):
        os.makedirs('build')

    output_path = os.path.join("build", exe_name)
    # 如果是 Windows 目标，加上 .exe 后缀
    if os_name == "windows" and not output_path.endswith(".exe"):
        output_path += ".exe"

    # 准备编译环境
    env = os.environ.copy()
    env['GOOS'] = os_name
    env['GOARCH'] = arch
    env['CGO_ENABLED'] = '0' # 静态编译

    # 编译命令
    cmd = (
        f'go build '
        f'-ldflags "-s -w -X swag-cli/internal/cli.Version={version}" '
        f'-o {output_path} '
        f'./cmd/swag-cli'
    )

    progress.set_status(f"正在编译 ({os_name}/{arch}) -> {output_path}...")
    run_cmd(cmd, progress, env=env)

    # 检查文件是否生成
    if not os.path.exists(output_path):
        progress.finish_step("❌ 编译失败: 未找到输出文件")
        sys.exit(1)

    return output_path


def deploy_to_remote(local_path: str, progress: ProgressDisplay) -> None:
    """部署到远程服务器"""
    host = str(get_config('REMOTE_HOST'))
    remote_temp = str(get_config('REMOTE_TEMP_PATH'))
    remote_install = str(get_config('REMOTE_INSTALL_PATH'))
    use_ssh_tty = bool(get_config('USE_SSH_TTY'))
    sudo_nopasswd = bool(get_config('SUDO_NOPASSWD'))

    filename = os.path.basename(local_path)
    remote_temp_file = f"{remote_temp}/{filename}"
    remote_target_file = f"{remote_install}/{filename}"

    # 1. SCP 上传到临时目录
    progress.set_status(f"📤 正在上传文件到 {host}:{remote_temp_file}...")
    scp_cmd = f"scp {local_path} {host}:{remote_temp_file}"
    run_cmd(scp_cmd, progress)

    # 2. 移动到安装目录并赋予权限
    progress.set_status(f"🔧 正在安装到 {remote_target_file}...")

    # 根据配置决定是否使用 -t 参数
    # 如果sudo已配置NOPASSWD，不需要TTY
    # 如果sudo需要密码且用户启用了USE_SSH_TTY，则使用 -t
    ssh_flags = ""
    if not sudo_nopasswd and use_ssh_tty:
        ssh_flags = "-t"

    # 使用 sudo 移动文件并设置权限
    install_cmd = (
        f'ssh {ssh_flags} {host} "sudo mv {remote_temp_file} {remote_target_file} && '
        f'sudo chmod +x {remote_target_file} && '
        f'ls -l {remote_target_file}"'
    )
    run_cmd(install_cmd, progress)


def main():
    progress = ProgressDisplay()
    print("🚀 开始构建和部署流程\n")

    # 1. 获取最新tag
    progress.set_status("🔍 获取最新tag...")
    version = get_latest_tag()
    progress.finish_step(f"✅ 最新tag: {version}")

    # 2. 编译
    progress.set_status("准备编译...")
    output_path = build_project(version, progress)
    progress.finish_step(f"✅ 编译完成: {output_path}")

    # 3. 部署
    if get_config('ENABLED_DEPLOY'):
        progress.set_status("🛰️  开始远程部署...")
        deploy_to_remote(output_path, progress)
        progress.finish_step("✅ 远程部署完成")
    else:
        print("\n⚠️  部署已禁用 (ENABLED_DEPLOY=False)，仅执行了编译。")

    print("\n🎉 所有任务已完成！")


if __name__ == "__main__":
    main()
