import { useState, type ReactNode } from 'react'
import { BadgeDollarSign, Pencil } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import { Button } from '../../../shared/ui/button'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import type { UpdateCallePayload } from '../api/update-calle'
import type { UpdateManzanaPayload } from '../api/update-manzana'
import type { UpdateCalleState } from '../hooks/use-update-calle'
import type { UpdateLoteState } from '../hooks/use-update-lote'
import type { UpdateManzanaState } from '../hooks/use-update-manzana'
import type { UpdateLotePayload } from '../lib/loteFormValues'
import type { LoteoDetail, PlanEntityRef } from '../types'
import CalleEditForm from './CalleEditForm'
import LoteEditForm from './LoteEditForm'
import ManzanaEditForm from './ManzanaEditForm'

const EMPTY_PROMPT = 'Tocá una manzana, un lote o una calle.'

type PlanSelectionPanelProps = {
  canEdit?: boolean
  selected: PlanEntityRef | null
  loteo: LoteoDetail
  polygonLabels: ReadonlyMap<string, string>
  selectedPolygonId: string | null
  updateState: UpdateLoteState
  onSave: (loteId: string, payload: UpdateLotePayload) => Promise<boolean>
  manzanaUpdateState: UpdateManzanaState
  onSaveManzana: (manzanaId: string, payload: UpdateManzanaPayload) => Promise<boolean>
  calleUpdateState: UpdateCalleState
  onSaveCalle: (calleId: string, payload: UpdateCallePayload) => Promise<boolean>
  renderReservationAction?: (lote: LoteoDetail['lotes'][number]) => ReactNode
  renderReservationSummary?: (lote: LoteoDetail['lotes'][number]) => ReactNode
}

export default function PlanSelectionPanel({
  canEdit = true,
  selected,
  loteo,
  polygonLabels,
  selectedPolygonId,
  updateState,
  onSave,
  manzanaUpdateState,
  onSaveManzana,
  calleUpdateState,
  onSaveCalle,
  renderReservationAction,
  renderReservationSummary,
}: PlanSelectionPanelProps) {
  const title = titleFor(selected, loteo, polygonLabels, selectedPolygonId)
  const selectedKey = selectionKey(selected)
  const [trackedSelectionKey, setTrackedSelectionKey] = useState(selectedKey)
  const [editingSelectionKey, setEditingSelectionKey] = useState<string | null>(null)

  if (selectedKey !== trackedSelectionKey) {
    setTrackedSelectionKey(selectedKey)
    setEditingSelectionKey(null)
  }

  if (selected === null || selected.kind === 'loteo') {
    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>Selección</CardTitle>
          <CardDescription>{EMPTY_PROMPT}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (selected.kind === 'lote') {
    const lote = loteo.lotes.find((item) => item.id === selected.id)
    if (!lote) {
      return (
        <Card size="sm">
          <CardHeader>
            <CardTitle>Selección</CardTitle>
            <CardDescription>{EMPTY_PROMPT}</CardDescription>
          </CardHeader>
        </Card>
      )
    }

    const isEditing = canEdit && editingSelectionKey === selectedKey
    const showEditAction = canEdit && !isEditing

    return (
      <Card size="sm" className="overflow-visible">
        <CardHeader>
          <CardTitle className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
            <span className="text-base font-semibold">{title}</span>
            <span className="text-xl font-semibold tabular-nums md:text-2xl">
              {lote.precio === null ? '—' : formatCurrency(lote.precio, lote.moneda)}
            </span>
          </CardTitle>
          <CardDescription className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
            <span>{loteMeta(lote, loteo)}</span>
            {pricePerSquareMeter(lote) && (
              <span className="tabular-nums">{pricePerSquareMeter(lote)}</span>
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {renderReservationSummary?.(lote)}

          {isEditing ? (
            <LoteEditForm
              lote={lote}
              updateState={updateState}
              onSave={(payload) => onSave(lote.id, payload)}
            />
          ) : (
            lote.caracteristicas && <p className="text-sm">{lote.caracteristicas}</p>
          )}

          {!isEditing && (
            <div className="flex flex-wrap gap-2">
              {renderReservationAction && lote.estado === 'disponible' && renderReservationAction(lote)}
              <Button type="button" variant="outline" disabled title="Próximamente">
                <BadgeDollarSign aria-hidden />
                Pasar a venta
              </Button>
              {showEditAction && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setEditingSelectionKey(selectedKey)}
                >
                  <Pencil aria-hidden />
                  Habilitar edición
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    )
  }

  if (selected.kind === 'manzana') {
    const manzana = loteo.manzanas.find((item) => item.id === selected.id)
    if (!manzana) {
      return (
        <Card size="sm">
          <CardHeader>
            <CardTitle>Selección</CardTitle>
            <CardDescription>{EMPTY_PROMPT}</CardDescription>
          </CardHeader>
        </Card>
      )
    }

    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>Número, servicios y calles de la manzana.</CardDescription>
        </CardHeader>
        <CardContent>
          {canEdit ? (
            <ManzanaEditForm
              manzana={manzana}
              calles={loteo.calles}
              loteCount={loteCountOf(loteo, manzana.id)}
              updateState={manzanaUpdateState}
              onSave={(payload) => onSaveManzana(manzana.id, payload)}
            />
          ) : (
            <ManzanaReadOnly
              manzana={manzana}
              calles={loteo.calles}
              loteCount={loteCountOf(loteo, manzana.id)}
            />
          )}
        </CardContent>
      </Card>
    )
  }

  const calle = loteo.calles.find((item) => item.id === selected.id)
  if (!calle) {
    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>Selección</CardTitle>
          <CardDescription>{EMPTY_PROMPT}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card size="sm">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>Nombre y tipo de la calle.</CardDescription>
      </CardHeader>
      <CardContent>
        {canEdit ? (
          <CalleEditForm
            calle={calle}
            updateState={calleUpdateState}
            onSave={(payload) => onSaveCalle(calle.id, payload)}
          />
        ) : (
          <CalleReadOnly calle={calle} />
        )}
      </CardContent>
    </Card>
  )
}

function loteMeta(lote: LoteoDetail['lotes'][number], loteo: LoteoDetail): string {
  const manzana = loteo.manzanas.find((item) => item.id === lote.manzanaId)
  const calles = (manzana?.calleIds ?? [])
    .map((id) => loteo.calles.find((calle) => calle.id === id)?.nombre)
    .filter((nombre): nombre is string => Boolean(nombre))

  return [
    manzana?.numero ? `Manzana ${manzana.numero}` : null,
    lote.superficie === null ? null : formatArea(lote.superficie),
    ...calles,
  ]
    .filter((part): part is string => part !== null)
    .join(' · ')
}

function pricePerSquareMeter(lote: LoteoDetail['lotes'][number]): string | null {
  if (lote.precio === null || lote.superficie === null || lote.superficie === 0) {
    return null
  }
  return `${formatCurrency(lote.precio / lote.superficie, lote.moneda)} por m²`
}

function ReadOnlyRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-border/60 pb-2 last:border-b-0 last:pb-0">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm">{value || '—'}</dd>
    </div>
  )
}

function ManzanaReadOnly({
  manzana,
  calles,
  loteCount,
}: {
  manzana: LoteoDetail['manzanas'][number]
  calles: LoteoDetail['calles']
  loteCount: number
}) {
  const services = [
    manzana.tieneAgua ? 'Agua' : null,
    manzana.tieneCloaca ? 'Cloaca' : null,
    manzana.tieneLuz ? 'Luz' : null,
    manzana.tieneGas ? 'Gas' : null,
  ].filter((service): service is string => service !== null)
  const calleNames = manzana.calleIds
    .map((id) => calles.find((calle) => calle.id === id)?.nombre)
    .filter((name): name is string => Boolean(name))

  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <ReadOnlyRow label="Número" value={manzana.numero} />
      <ReadOnlyRow label="Lotes" value={String(loteCount)} />
      <ReadOnlyRow
        label="Servicios"
        value={services.length > 0 ? services.join(', ') : 'Ninguno'}
      />
      <ReadOnlyRow
        label="Calles"
        value={calleNames.length > 0 ? calleNames.join(', ') : 'Ninguna'}
      />
    </dl>
  )
}

function CalleReadOnly({ calle }: { calle: LoteoDetail['calles'][number] }) {
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <ReadOnlyRow label="Nombre" value={calle.nombre} />
      <ReadOnlyRow label="Tipo" value={calle.tipo} />
    </dl>
  )
}

function selectionKey(selected: PlanEntityRef | null): string {
  if (selected === null) {
    return ''
  }
  if (selected.kind === 'loteo') {
    return 'loteo'
  }
  return `${selected.kind}:${selected.id}`
}

function loteCountOf(loteo: LoteoDetail, manzanaId: string): number {
  return loteo.lotes.filter((lote) => lote.manzanaId === manzanaId).length
}

function titleFor(
  selected: PlanEntityRef | null,
  loteo: LoteoDetail,
  polygonLabels: ReadonlyMap<string, string>,
  selectedPolygonId: string | null,
): string {
  if (selectedPolygonId) {
    const label = polygonLabels.get(selectedPolygonId)
    if (label) {
      return label
    }
  }
  if (selected?.kind === 'lote') {
    const lote = loteo.lotes.find((item) => item.id === selected.id)
    return lote?.numero ? `Lote ${lote.numero}` : 'Lote'
  }
  if (selected?.kind === 'manzana') {
    const manzana = loteo.manzanas.find((item) => item.id === selected.id)
    return manzana?.numero ? `Manzana ${manzana.numero}` : 'Manzana'
  }
  if (selected?.kind === 'calle') {
    const calle = loteo.calles.find((item) => item.id === selected.id)
    return calle?.nombre ? `Calle ${calle.nombre}` : 'Calle'
  }
  return 'Selección'
}
