const platformDecisions = [
  '移动端 H5 网页',
  '注册后参与',
  '仅使用虚拟金币',
  '不充值、不提现、不交易',
]

export function App() {
  return (
    <main className="shell">
      <section className="card" aria-labelledby="page-title">
        <div className="brand-mark" aria-hidden="true">刮</div>
        <p className="eyebrow">首款游戏 · 产品筹备中</p>
        <h1 id="page-title">刮刮乐游戏平台</h1>
        <p className="summary">
          工程基础已经就绪。下一步将共同确定第一款刮刮乐的主题、图案、奖项结构与完整游玩流程。
        </p>

        <ul className="decision-list" aria-label="已确定的产品方向">
          {platformDecisions.map((decision) => (
            <li key={decision}>{decision}</li>
          ))}
        </ul>

        <div className="status" role="status">
          <span className="status-dot" aria-hidden="true" />
          开发环境已准备
        </div>
      </section>
    </main>
  )
}

