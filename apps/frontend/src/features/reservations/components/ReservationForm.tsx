import { useState, type FormEvent, type ReactNode } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Field, FieldDescription, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { newIdempotencyKey } from '../../../shared/lib/idempotencyKey'
import {
  DEFAULT_RESERVATION_VALIDITY_DAYS,
  MAX_RESERVATION_VALIDITY_DAYS,
  type ReservationClient,
  type ReservationCreateValues,
  type ReservationDraft,
  type ReservationLot,
  type ReservationLoteoOption,
  type SellerOption,
} from '../types'

type Props = {
  loteos: ReservationLoteoOption[]
  selectedLoteoId: string
  selectedLoteId: string
  lots: ReservationLot[]
  clients: ReservationClient[]
  sellers: SellerOption[]
  isLoadingSellers: boolean
  isSubmitting: boolean
  error: string | null
  fixedTarget?: boolean
  disabled?: boolean
  onLoteoChange?: (value: string) => void
  onLoteChange?: (value: string) => void
  onReset?: () => void
  onSubmit: (values: ReservationCreateValues, key: string) => Promise<boolean>
  onCancel?: () => void
  idempotencyKey?: string
  onCompleted?: () => void
  draft?: ReservationDraft
  onDraftChange?: (draft: ReservationDraft) => void
  sellerIsFixed?: boolean
  fixedSeller?: SellerOption
  sellerRequired?: boolean
  canSetValidity?: boolean
  renderClientAction?: ReactNode
}

export default function ReservationForm({
  loteos,
  selectedLoteoId,
  selectedLoteId,
  lots,
  clients,
  sellers,
  isLoadingSellers,
  isSubmitting,
  error,
  fixedTarget = false,
  disabled = false,
  onLoteoChange,
  onLoteChange,
  onReset,
  onSubmit,
  onCancel,
  idempotencyKey,
  onCompleted,
  draft,
  onDraftChange,
  sellerIsFixed = false,
  fixedSeller,
  sellerRequired = sellers.length > 1,
  canSetValidity = false,
  renderClientAction,
}: Props) {
  const [localClienteId, setLocalClienteId] = useState('')
  const [localVendedorId, setLocalVendedorId] = useState('')
  const [validityDays, setValidityDays] = useState(String(DEFAULT_RESERVATION_VALIDITY_DAYS))
  const [localIdempotencyKey, setLocalIdempotencyKey] = useState(newIdempotencyKey)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const selectedLote = lots.find((lot) => lot.id === selectedLoteId)
  const clienteId = draft?.clienteId ?? localClienteId
  const vendedorId = draft?.vendedorId ?? localVendedorId

  function setClienteId(value: string) {
    if (draft && onDraftChange) onDraftChange({ ...draft, clienteId: value })
    else setLocalClienteId(value)
  }

  function setVendedorId(value: string) {
    if (draft && onDraftChange) onDraftChange({ ...draft, vendedorId: value })
    else setLocalVendedorId(value)
  }

  function clearFieldError(field: string) {
    setFieldErrors((current) => {
      if (!(field in current)) return current
      const next = { ...current }
      delete next[field]
      return next
    })
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nextErrors: Record<string, string> = {}
    if (!selectedLoteoId) nextErrors.loteo = 'Seleccioná un loteo.'
    if (!selectedLoteId) nextErrors.lote = 'Seleccioná un lote.'
    if (!clienteId) nextErrors.cliente = 'Seleccioná un cliente.'
    const selectedSellerId = sellerIsFixed ? fixedSeller?.id ?? sellers[0]?.id ?? '' : vendedorId
    if (sellerRequired && !selectedSellerId) nextErrors.vendedor = 'Seleccioná un vendedor.'
    const diasVigencia = Number(validityDays.trim())
    if (canSetValidity && !isValidValidity(validityDays, diasVigencia)) {
      nextErrors.vigencia = `Ingresá una vigencia entre 1 y ${MAX_RESERVATION_VALIDITY_DAYS} días.`
    }
    setFieldErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    const values: ReservationCreateValues = { loteoId: selectedLoteoId, loteId: selectedLoteId, clienteId, vendedorId: selectedSellerId || undefined }
    if (canSetValidity) values.diasVigencia = diasVigencia
    const completed = await onSubmit(values, idempotencyKey ?? localIdempotencyKey)
    if (completed) {
      if (idempotencyKey === undefined) setLocalIdempotencyKey(newIdempotencyKey())
      setClienteId('')
      setVendedorId('')
      setValidityDays(String(DEFAULT_RESERVATION_VALIDITY_DAYS))
      setFieldErrors({})
      onReset?.()
      onCompleted?.()
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Nueva reserva</CardTitle>
        <CardDescription>
          {canSetValidity
            ? `El plazo se cuenta desde la confirmación del servidor; por defecto es de ${DEFAULT_RESERVATION_VALIDITY_DAYS} días.`
            : 'El plazo es de 360 horas desde la confirmación del servidor.'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="grid gap-4" onSubmit={handleSubmit} noValidate>
          {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
          {!fixedTarget && (
            <div className="grid gap-4 sm:grid-cols-2">
              <ReservationSelectField
                id="reserva-loteo"
                label="Loteo"
                value={selectedLoteoId}
                placeholder="Seleccioná un loteo"
                options={loteos.map((loteo) => ({ value: loteo.id, label: loteo.nombre }))}
                onChange={onLoteoChange!}
                error={fieldErrors.loteo}
              />
              <ReservationSelectField
                id="reserva-lote"
                label="Lote"
                value={selectedLoteId}
                placeholder={lots.length ? 'Seleccioná un lote' : 'No hay lotes disponibles'}
                options={lots.map((lot) => ({ value: lot.id, label: lot.numero ? `Lote ${lot.numero}` : 'Lote sin número' }))}
                onChange={onLoteChange!}
                error={fieldErrors.lote}
                disabled={lots.length === 0}
              />
            </div>
          )}
          {fixedTarget && selectedLote && (
            <div className="rounded-lg border border-border bg-muted/30 p-3 text-sm">
              <p className="font-medium">{selectedLote.numero ? `Lote ${selectedLote.numero}` : 'Lote seleccionado'}</p>
              <p className="text-muted-foreground">El loteo seleccionado y su disponibilidad se validan nuevamente al guardar.</p>
            </div>
          )}
          <ReservationSelectField
            id="reserva-cliente"
            label="Cliente"
            value={clienteId}
            placeholder={clients.length ? 'Seleccioná un cliente' : 'No hay clientes activos'}
            options={clients.map((client) => ({ value: client.id, label: `${client.apellido}, ${client.nombre} · DNI ${client.dni}` }))}
            onChange={(value) => { setClienteId(value); clearFieldError('cliente') }}
            error={fieldErrors.cliente}
            disabled={clients.length === 0}
            action={renderClientAction}
            onOpenChange={(open) => { if (open) clearFieldError('cliente') }}
          />
          {sellerIsFixed ? (
            <Field>
              <FieldLabel>Vendedor</FieldLabel>
              <div className="rounded-md border border-border bg-muted/30 px-3 py-2 text-sm">
                {fixedSeller ? `${fixedSeller.apellido}, ${fixedSeller.nombre}` : isLoadingSellers ? 'Cargando vendedor…' : 'No hay un vendedor habilitado'}
              </div>
            </Field>
          ) : (
            <ReservationSelectField
              id="reserva-vendedor"
              label="Vendedor"
              value={vendedorId}
              placeholder={isLoadingSellers ? 'Cargando vendedores…' : 'Seleccioná un vendedor'}
              options={sellers.map((seller) => ({ value: seller.id, label: `${seller.apellido}, ${seller.nombre}` }))}
              onChange={(value) => { setVendedorId(value); clearFieldError('vendedor') }}
              error={fieldErrors.vendedor}
              disabled={isLoadingSellers || sellers.length === 0}
              onOpenChange={(open) => { if (open) clearFieldError('vendedor') }}
            />
          )}
          {canSetValidity && (
            <Field data-invalid={Boolean(fieldErrors.vigencia)}>
              <FieldLabel htmlFor="reserva-vigencia">Vigencia (días)</FieldLabel>
              <Input
                id="reserva-vigencia"
                name="diasVigencia"
                inputMode="numeric"
                autoComplete="off"
                value={validityDays}
                onChange={(event) => { setValidityDays(event.target.value); clearFieldError('vigencia') }}
                aria-invalid={Boolean(fieldErrors.vigencia)}
                aria-describedby={fieldErrors.vigencia ? 'reserva-vigencia-error' : 'reserva-vigencia-description'}
                className="min-h-11 md:min-h-9"
              />
              <FieldDescription id="reserva-vigencia-description">
                Entre 1 y {MAX_RESERVATION_VALIDITY_DAYS} días.
              </FieldDescription>
              <FieldError id="reserva-vigencia-error">{fieldErrors.vigencia}</FieldError>
            </Field>
          )}
          <p className="text-sm text-muted-foreground">La fecha de vencimiento se calcula con la hora oficial del servidor.</p>
          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            {onCancel && <Button type="button" variant="outline" onClick={onCancel}>Cancelar</Button>}
            <Button type="submit" disabled={isSubmitting || disabled || clients.length === 0 || sellers.length === 0}>
              {isSubmitting ? 'Guardando…' : 'Confirmar reserva'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function isValidValidity(raw: string, days: number): boolean {
  return raw.trim() !== '' && Number.isInteger(days) && days >= 1 && days <= MAX_RESERVATION_VALIDITY_DAYS
}

function ReservationSelectField({
  id,
  label,
  value,
  placeholder,
  options,
  onChange,
  error,
  disabled = false,
  action,
  onOpenChange,
}: {
  id: string
  label: string
  value: string
  placeholder: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
  error?: string
  disabled?: boolean
  action?: ReactNode
  onOpenChange?: (open: boolean) => void
}) {
  const errorId = `${id}-error`
    const emptyValue = `${id}-empty`
  return (
    <Field data-invalid={Boolean(error)}>
      {action ? (
        <div className="flex items-center justify-between gap-3">
          <FieldLabel htmlFor={id}>{label}</FieldLabel>
          {action}
        </div>
      ) : <FieldLabel htmlFor={id}>{label}</FieldLabel>}
      <Select
        value={value || emptyValue}
        onOpenChange={onOpenChange}
        onValueChange={(nextValue) => {
          const selectedValue = nextValue ?? emptyValue
          onChange(selectedValue === emptyValue ? '' : selectedValue)
        }}
        disabled={disabled}
      >
        <SelectTrigger id={id} aria-invalid={Boolean(error)} aria-describedby={error ? errorId : undefined}>
          <SelectValue>
            {value ? options.find((option) => option.value === value)?.label : placeholder}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectList>
            <SelectItem value={emptyValue}>{placeholder}</SelectItem>
            {options.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}
          </SelectList>
        </SelectContent>
      </Select>
      <FieldError id={errorId}>{error}</FieldError>
    </Field>
  )
}
