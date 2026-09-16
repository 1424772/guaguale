import { PointerEvent, RefObject, useEffect, useState } from 'react'
import type { Ticket } from './api'
import { ticketArtworkFor } from './ticketArtwork'

export type DeskPlacement = {
  deskX: number
  deskY: number
  rotation: number
  zIndex: number
}

type DeskTicketProps = {
  ticket: Ticket
  deskRef: RefObject<HTMLDivElement | null>
  topZ: number
  onOpen: (ticket: Ticket) => void
  onPin: (ticket: Ticket) => void
  onRobot: (ticket: Ticket) => void
  fanAction?: 'discarded' | 'robot' | 'caught' | 'safe'
  motion?: 'robot-ejected' | 'redeeming' | 'discarding'
  onDrop: (ticket: Ticket, point: { x: number; y: number }, placement: DeskPlacement) => void
}

const clamp = (value: number, minimum: number, maximum: number) => Math.max(minimum, Math.min(maximum, value))

export function DeskTicket({ ticket, deskRef, topZ, onOpen, onPin, onRobot, fanAction, motion, onDrop }: DeskTicketProps) {
  const [placement, setPlacement] = useState<DeskPlacement>({
    deskX: ticket.deskX,
    deskY: ticket.deskY,
    rotation: ticket.rotation,
    zIndex: ticket.zIndex,
  })
  const [drag, setDrag] = useState<{
    pointerId: number
    startX: number
    startY: number
    originX: number
    originY: number
    moved: boolean
  } | null>(null)

  useEffect(() => {
    if (drag) return
    setPlacement({ deskX: ticket.deskX, deskY: ticket.deskY, rotation: ticket.rotation, zIndex: ticket.zIndex })
  }, [drag, ticket.deskX, ticket.deskY, ticket.rotation, ticket.zIndex])

  function pointerMove(event: PointerEvent<HTMLElement>) {
    if (!drag || drag.pointerId !== event.pointerId) return
    const bounds = deskRef.current?.getBoundingClientRect()
    if (!bounds) return
    const deltaX = (event.clientX - drag.startX) / bounds.width
    const deltaY = (event.clientY - drag.startY) / bounds.height
    const moved = drag.moved || Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY) > 5
    setDrag({ ...drag, moved })
    setPlacement((current) => ({
      ...current,
      deskX: clamp(drag.originX + deltaX, .08, .92),
      deskY: clamp(drag.originY + deltaY, .09, .9),
    }))
  }

  function pointerEnd(event: PointerEvent<HTMLElement>) {
    if (!drag || drag.pointerId !== event.pointerId) return
    const wasMoved = drag.moved
    setDrag(null)
    if (wasMoved) {
      onDrop(ticket, { x: event.clientX, y: event.clientY }, placement)
    } else {
      onOpen(ticket)
    }
  }

  return (
    <article
      className={`movable-ticket state-${ticket.state} card-${ticket.cardCode} ${drag?.moved ? 'dragging' : ''} ${fanAction ? `fan-${fanAction}` : ''} ${motion ?? ''}`}
      style={{
        left: `${placement.deskX * 100}%`,
        top: `${placement.deskY * 100}%`,
        zIndex: placement.zIndex,
        transform: `translate(-50%, -50%) rotate(${placement.rotation}deg)`,
      }}
      role="button"
      tabIndex={0}
      aria-label={`${ticket.cardName}，${ticket.state === 'purchased' ? '等待刮开' : ticket.reward ? `中奖${ticket.reward}金币` : '未中奖'}`}
      onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') onOpen(ticket) }}
      onPointerDown={(event) => {
        const nextZ = Math.max(topZ + 1, ticket.zIndex)
        setPlacement((current) => ({ ...current, zIndex: nextZ }))
        setDrag({
          pointerId: event.pointerId,
          startX: event.clientX,
          startY: event.clientY,
          originX: placement.deskX,
          originY: placement.deskY,
          moved: false,
        })
        event.currentTarget.setPointerCapture(event.pointerId)
      }}
      onPointerMove={pointerMove}
      onPointerUp={pointerEnd}
      onPointerCancel={() => setDrag(null)}
    >
      <img className="ticket-artwork" src={ticketArtworkFor(ticket.cardCode)} alt="" draggable={false} />
      <span className="movable-ticket-name">{ticket.cardName}</span>
      <div className="movable-ticket-scratch">
        {ticket.state === 'purchased' ? <span>刮奖区</span> : <span>{ticket.reward ? `中奖 ${ticket.reward}` : '未中奖'}</span>}
      </div>
      <small>{ticket.source === 'daily_wheel' ? '每日转盘赠送' : `${ticket.nominalPrice} 金币`}</small>
      <button
        type="button"
        className="ticket-pin"
        aria-label={`将${ticket.cardName}放入固定卡槽`}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={(event) => {
          event.stopPropagation()
          onPin(ticket)
        }}
      >固定</button>
      {ticket.state === 'purchased' && <button
        type="button"
        className={`ticket-robot ${ticket.cardCode === 'all-in' ? 'disabled' : ''}`}
        aria-label={`将${ticket.cardName}交给机器人`}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={(event) => {
          event.stopPropagation()
          if (ticket.cardCode !== 'all-in') onRobot(ticket)
        }}
      >{ticket.cardCode === 'all-in' ? '仅手动' : '机器人'}</button>}
      <i className="drag-grip" aria-hidden="true">⠿</i>
    </article>
  )
}
