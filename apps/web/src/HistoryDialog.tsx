import { useMemo, useState } from 'react'
import type { HistoryEvent } from './api'

const coinFormatter = new Intl.NumberFormat('zh-CN')
const filters: Array<{ code: 'all' | HistoryEvent['type']; label: string }> = [
  { code: 'all', label: '全部' }, { code: 'purchase', label: '获得/购买' }, { code: 'scratch', label: '手动刮奖' },
  { code: 'robot', label: '机器人' }, { code: 'redeem', label: '兑奖' }, { code: 'discard', label: '丢弃' },
]
const icons: Record<HistoryEvent['type'], string> = { purchase: '🎟', scratch: '✦', robot: '▣', redeem: '◆', discard: '⌫' }

type Props = {
  events: HistoryEvent[]
  loading: boolean
  error: string
  onClose: () => void
  onRefresh: () => void
}

export function HistoryDialog({ events, loading, error, onClose, onRefresh }: Props) {
  const [filter, setFilter] = useState<'all' | HistoryEvent['type']>('all')
  const visible = useMemo(() => filter === 'all' ? events : events.filter((event) => event.type === filter), [events, filter])
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose() }}>
      <section className="history-dialog" role="dialog" aria-modal="true" aria-label="游戏记录">
        <button className="close-button" type="button" onClick={onClose} aria-label="关闭">×</button>
        <header className="history-header">
          <div><span className="eyebrow">ACTIVITY ARCHIVE</span><h1>游戏记录</h1><p>永久保留最近的购卡、刮奖、兑奖与丢弃结果。</p></div>
          <button type="button" className="secondary-button" onClick={onRefresh} disabled={loading}>刷新</button>
        </header>
        <nav className="history-filters" aria-label="记录筛选">
          {filters.map((item) => <button type="button" className={filter === item.code ? 'active' : ''} onClick={() => setFilter(item.code)} key={item.code}>{item.label}</button>)}
        </nav>
        {loading && events.length === 0 ? <div className="dialog-loading"><span className="spinner" />正在整理记录…</div> : error ? (
          <div className="dialog-error"><strong>记录暂时无法读取</strong><span>{error}</span><button type="button" onClick={onRefresh}>重新加载</button></div>
        ) : visible.length === 0 ? <div className="history-empty"><span>◇</span><strong>这个分类还没有记录</strong><small>完成一次购卡或刮奖后会显示在这里</small></div> : (
          <div className="history-timeline">
            {visible.map((event) => (
              <article className={`history-event event-${event.type}`} key={event.id}>
                <div className="history-icon" aria-hidden="true">{icons[event.type]}</div>
                <div><span>{event.title}{event.automated && <small>自动</small>}</span><strong>{event.detail}</strong><time>{new Date(event.createdAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })}</time></div>
                {Boolean(event.delta) && <b className={(event.delta ?? 0) > 0 ? 'positive' : 'negative'}>{(event.delta ?? 0) > 0 ? '+' : ''}{coinFormatter.format(event.delta ?? 0)}</b>}
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
