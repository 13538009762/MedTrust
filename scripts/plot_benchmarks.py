#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
MedTrust 毕业论文第 5、6 章性能实验图表绘制脚本
读取或使用 benchmark_test.go 输出的测试指标，自动绘制 3 张高清晰度学术论文级图表:
1. 图 5-1: 不同医疗文件尺寸下的数据加密、哈希计算与 IPFS 存取全链路开销对比
2. 图 5-2: 三级零信任访问控制与风险引擎在不同并发度下的吞吐量 (TPS) 与延迟 (ms)
3. 图 5-3: 智能合约 4 组核心存证交易平均延迟与 P95 分位延迟对比
"""

import os
import sys

def main():
    try:
        import matplotlib.pyplot as plt
        import matplotlib
        import numpy as np
    except ImportError:
        print("[PlotScript] 提示: 请先安装绘图依赖: pip install matplotlib numpy")
        sys.exit(0)

    # 设置学术论文风格字体 (支持跨平台中文回退)
    plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'DejaVu Sans', 'Arial']
    plt.rcParams['axes.unicode_minus'] = False
    plt.rcParams['figure.dpi'] = 300

    output_dir = os.path.join(os.path.dirname(__file__), "..", "docs", "figures")
    os.makedirs(output_dir, exist_ok=True)

    # -------------------------------------------------------------------------
    # 图 5-1: 医疗文件尺寸 vs 全链路处理耗时 (堆叠/分组柱状图)
    # -------------------------------------------------------------------------
    sizes = ['100 KB', '500 KB', '1 MB', '5 MB', '10 MB']
    sha256_times = [0.05, 0.55, 1.22, 4.31, 7.45]
    aes_enc_times = [0.10, 0.54, 2.57, 6.72, 7.15]
    ipfs_write_times = [0.52, 1.24, 2.01, 3.93, 8.34]
    ipfs_read_times = [17.21, 21.80, 17.12, 23.93, 17.06]
    aes_dec_times = [0.05, 0.20, 0.78, 2.50, 5.08]

    x = np.arange(len(sizes))
    width = 0.55

    fig, ax = plt.subplots(figsize=(8, 5))
    p1 = ax.bar(x, sha256_times, width, label='SHA-256 计算', color='#3498db')
    p2 = ax.bar(x, aes_enc_times, width, bottom=sha256_times, label='AES-256-GCM 加密', color='#2ecc71')
    bottom_3 = np.array(sha256_times) + np.array(aes_enc_times)
    p3 = ax.bar(x, ipfs_write_times, width, bottom=bottom_3, label='IPFS 块写入', color='#e67e22')
    bottom_4 = bottom_3 + np.array(ipfs_write_times)
    p4 = ax.bar(x, ipfs_read_times, width, bottom=bottom_4, label='IPFS 块读取', color='#9b59b6')
    bottom_5 = bottom_4 + np.array(ipfs_read_times)
    p5 = ax.bar(x, aes_dec_times, width, bottom=bottom_5, label='AES-256-GCM 解密', color='#e74c3c')

    ax.set_ylabel('处理耗时 (毫秒 / ms)', fontsize=11)
    ax.set_xlabel('电子病历及影像附件文件尺寸', fontsize=11)
    ax.set_title('图 5-1 不同医疗数据文件尺寸下的全链路密码与存储开销对比', fontsize=12, fontweight='bold', pad=12)
    ax.set_xticks(x)
    ax.set_xticklabels(sizes, fontsize=10)
    ax.legend(loc='upper left', frameon=True)
    ax.grid(axis='y', linestyle='--', alpha=0.5)

    fig.tight_layout()
    fig1_path = os.path.join(output_dir, "fig5_1_crypto_storage_latency.png")
    fig.savefig(fig1_path)
    plt.close(fig)
    print(f"[PlotScript] ✅ 图 5-1 已生成并保存至: {fig1_path}")

    # -------------------------------------------------------------------------
    # 图 5-2: 访问控制引擎高并发吞吐量与平均延迟 (双 Y 轴折线+柱状图)
    # -------------------------------------------------------------------------
    concurrencies = [50, 100, 200, 500, 800]
    tps = [5741.5, 9073.7, 9268.7, 9632.2, 9294.4]
    avg_latency = [7.14, 7.73, 12.66, 34.91, 60.15]
    p95_latency = [17.60, 14.03, 18.49, 47.76, 80.03]

    fig, ax1 = plt.subplots(figsize=(8, 5))

    color_tps = '#2980b9'
    ax1.set_xlabel('并发线程数 (Concurrency Level)', fontsize=11)
    ax1.set_ylabel('系统吞吐量 (TPS / 事务每秒)', color=color_tps, fontsize=11)
    line1 = ax1.plot(concurrencies, tps, 'o-', color=color_tps, linewidth=2.5, markersize=7, label='系统吞吐量 (TPS)')
    ax1.tick_params(axis='y', labelcolor=color_tps)
    ax1.grid(True, linestyle='--', alpha=0.4)

    ax2 = ax1.twinx()
    color_lat = '#e74c3c'
    ax2.set_ylabel('判定响应延迟 (毫秒 / ms)', color=color_lat, fontsize=11)
    line2 = ax2.plot(concurrencies, avg_latency, 's--', color=color_lat, linewidth=2, markersize=6, label='平均判定延迟 (Avg)')
    line3 = ax2.plot(concurrencies, p95_latency, '^:', color='#d35400', linewidth=1.8, markersize=6, label='P95 延迟')
    ax2.tick_params(axis='y', labelcolor=color_lat)

    lines = line1 + line2 + line3
    labels = [l.get_label() for l in lines]
    ax1.legend(lines, labels, loc='center right', frameon=True)

    plt.title('图 5-2 三级访问控制与风险判定引擎在高并发下的吞吐量与延迟曲线', fontsize=12, fontweight='bold', pad=12)
    fig.tight_layout()
    fig2_path = os.path.join(output_dir, "fig5_2_concurrency_tps_latency.png")
    fig.savefig(fig2_path)
    plt.close(fig)
    print(f"[PlotScript] ✅ 图 5-2 已生成并保存至: {fig2_path}")

    # -------------------------------------------------------------------------
    # 图 5-3: 智能合约 4 组核心存证调用延迟对比 (柱状图与 P95 误差线)
    # -------------------------------------------------------------------------
    ops = ['CreateMedicalAsset\n(病历存证)', 'CreateAuthorization\n(患者授权)', 'CreateEmergency\n(破窗应急)', 'QueryMedicalAsset\n(链上溯源)']
    avg_lats = [18.53, 24.59, 23.06, 0.05]
    p95_lats = [30.65, 32.09, 32.23, 0.10]
    p50_lats = [21.84, 24.46, 22.96, 0.02]

    x = np.arange(len(ops))
    width = 0.35

    fig, ax = plt.subplots(figsize=(8.5, 4.8))
    rects1 = ax.bar(x - width/2, avg_lats, width, label='平均延迟 (Avg)', color='#1abc9c')
    rects2 = ax.bar(x + width/2, p95_lats, width, label='P95 峰值延迟', color='#f39c12')

    ax.set_ylabel('响应延迟 (毫秒 / ms)', fontsize=11)
    ax.set_title('图 5-3 智能合约 4 组核心存证与状态查询调用延迟分布对比', fontsize=12, fontweight='bold', pad=12)
    ax.set_xticks(x)
    ax.set_xticklabels(ops, fontsize=10)
    ax.legend(frameon=True)
    ax.grid(axis='y', linestyle='--', alpha=0.5)

    for rect in rects1:
        height = rect.get_height()
        ax.annotate(f'{height:.1f}',
                    xy=(rect.get_x() + rect.get_width() / 2, height),
                    xytext=(0, 3), textcoords="offset points",
                    ha='center', va='bottom', fontsize=9)

    for rect in rects2:
        height = rect.get_height()
        ax.annotate(f'{height:.1f}',
                    xy=(rect.get_x() + rect.get_width() / 2, height),
                    xytext=(0, 3), textcoords="offset points",
                    ha='center', va='bottom', fontsize=9)

    fig.tight_layout()
    fig3_path = os.path.join(output_dir, "fig5_3_contract_latency.png")
    fig.savefig(fig3_path)
    plt.close(fig)
    print(f"[PlotScript] ✅ 图 5-3 已生成并保存至: {fig3_path}")

    print("\n🎉 毕业论文第 5、6 章全部 3 张实验图表已成功生成完毕！可在 docs/figures/ 目录下查阅并直接插入 Word/LaTeX 论文正文。")

if __name__ == '__main__':
    main()
