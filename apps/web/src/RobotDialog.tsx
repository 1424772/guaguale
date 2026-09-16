import type { RobotStatus } from './api'

type Props = {
  robot: RobotStatus
  onClose: () => void
  onOpenShop: (code: 'robot-speed' | 'robot-queue' | 'robot-intercept') => void
}

function seconds(milliseconds: number) {
  return Math.max(0, Math.ceil(milliseconds / 1000))
}

export function RobotDialog({ robot, onClose, onOpenShop }: Props) {
  const active = robot.queue[0]
  const progress = active ? Math.max(0, Math.min(100, 100 - active.remainingMs / (robot.durationSeconds * 10))) : 0
  return (
    <div className="modal-backdrop robot-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose()
    }}>
      <section className="robot-dialog" role="dialog" aria-modal="true" aria-labelledby="robot-title">
        <button className="close-button" type="button" onClick={onClose} aria-label="关闭机器人面板">×</button>
        <header><span className="eyebrow">AUTO SCRATCH UNIT</span><h1 id="robot-title">自动刮奖机器人</h1><p>页面可见时运行，关闭或切到后台会暂停。</p></header>
        <div className="robot-current">
          <div className={`robot-machine-large ${active ? 'working' : ''}`}><span>▣</span><i /><i /><i /></div>
          <div>
            <small>{active ? '正在处理' : '当前状态'}</small>
            <h2>{active?.cardName ?? '等待刮刮乐'}</h2>
            <p>{active ? `剩余约 ${seconds(active.remainingMs)} 秒` : '把未刮开的卡拖到机器人处即可入队。'}</p>
            <div className="robot-progress"><span style={{ width: `${progress}%` }} /></div>
          </div>
        </div>
        <div className="robot-modules">
          <button type="button" onClick={() => onOpenShop('robot-speed')}><small>速度等级 {robot.speedLevel}/8</small><strong>{robot.durationSeconds}秒/张</strong><span>升级 →</span></button>
          <button type="button" onClick={() => onOpenShop('robot-queue')}><small>队列等级 {robot.queueLevel}/6</small><strong>{robot.capacity}张容量</strong><span>升级 →</span></button>
          <button type="button" onClick={() => onOpenShop('robot-intercept')}><small>拦截等级 {robot.interceptLevel}/8</small><strong>{robot.interceptPercent}%</strong><span>升级 →</span></button>
        </div>
        <div className="robot-queue-list">
          <div><strong>等待队列</strong><span>{robot.queue.length} / {robot.capacity}</span></div>
          {robot.queue.length === 0 ? <p>队列为空</p> : robot.queue.map((item) => (
            <article key={item.ticketId} className={item.position === 1 ? 'active' : ''}><span>{item.position}</span><strong>{item.cardName}</strong><small>{item.position === 1 ? `${seconds(item.remainingMs)}秒` : '等待中'}</small></article>
          ))}
        </div>
      </section>
    </div>
  )
}
