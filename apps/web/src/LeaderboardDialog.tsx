import type { Leaderboard } from './api'

const coinFormatter = new Intl.NumberFormat('zh-CN')

type Props = {
  leaderboard: Leaderboard | null
  loading: boolean
  error: string
  onClose: () => void
  onRefresh: () => void
}

export function LeaderboardDialog({ leaderboard, loading, error, onClose, onRefresh }: Props) {
  const podium = leaderboard?.entries.slice(0, 3) ?? []
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose() }}>
      <section className="ranking-dialog" role="dialog" aria-modal="true" aria-label="财富排行榜">
        <button className="close-button" type="button" onClick={onClose} aria-label="关闭">×</button>
        <header className="ranking-header">
          <div><span className="eyebrow">WEALTH RANKING</span><h1>财富排行榜</h1><p>只按当前剩余金币排名，不发放额外奖励。</p></div>
          <button type="button" className="secondary-button" onClick={onRefresh} disabled={loading}>刷新</button>
        </header>

        {loading && !leaderboard ? <div className="dialog-loading"><span className="spinner" />正在读取排名…</div> : error ? (
          <div className="dialog-error"><strong>排行榜暂时无法读取</strong><span>{error}</span><button type="button" onClick={onRefresh}>重新加载</button></div>
        ) : leaderboard && (
          <>
            <div className="ranking-podium">
              {podium.map((player, index) => (
                <article className={`podium-place place-${index + 1} ${player.isCurrent ? 'current' : ''}`} key={`${player.rank}-${player.username}`}>
                  <span>{index === 0 ? '♛' : index === 1 ? '◆' : '●'}</span>
                  <small>第 {player.rank} 名</small>
                  <strong>{player.username}</strong>
                  <b>{coinFormatter.format(player.balance)} 金币</b>
                </article>
              ))}
            </div>

            <div className="ranking-list" role="table" aria-label="排行榜前100名">
              <div className="ranking-row ranking-labels" role="row"><span>名次</span><span>玩家</span><span>剩余金币</span></div>
              {leaderboard.entries.map((player) => (
                <div className={`ranking-row ${player.isCurrent ? 'current' : ''}`} role="row" key={`${player.rank}-${player.username}`}>
                  <span>{player.rank <= 3 ? ['🥇', '🥈', '🥉'][player.rank - 1] : `#${player.rank}`}</span>
                  <strong>{player.username}{player.isCurrent && <small>你</small>}</strong>
                  <b>{coinFormatter.format(player.balance)}</b>
                </div>
              ))}
            </div>

            <footer className="my-ranking">
              <span>我的实时排名</span><strong>#{leaderboard.currentUser.rank}</strong><b>{coinFormatter.format(leaderboard.currentUser.balance)} 金币</b>
              <small>共 {leaderboard.totalUsers} 位玩家 · 榜单快照 {new Date(leaderboard.generatedAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</small>
            </footer>
          </>
        )}
      </section>
    </div>
  )
}
