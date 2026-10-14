import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import {
  ReceiptDialog,
  ReceiptField,
  ReceiptFields,
  ReceiptSection,
  ReceiptSignatures,
  ReceiptSummary,
  ReceiptTable,
  ReceiptTableCell,
  ReceiptTableRow,
} from './receipt'

function renderReceipt(props: { open?: boolean; onClose?: () => void } = {}) {
  return render(
    <ReceiptDialog
      open={props.open ?? true}
      title="Recibo de prueba"
      description="Revisá el recibo."
      documentTitle="Comprobante de prueba"
      issuedAt="2026-09-07T10:00:00.000Z"
      printLabel="Imprimir comprobante"
      onClose={props.onClose ?? vi.fn()}
    >
      <ReceiptSummary label="Total" asideLabel="Modalidad" aside="Contado">
        US$ 1.000,00
      </ReceiptSummary>
      <ReceiptSection title="Datos del comprobante">
        <ReceiptFields label="Datos">
          <ReceiptField term="Cliente">Pérez, Ana</ReceiptField>
          <ReceiptField term="Observación" wide>
            Sin observaciones
          </ReceiptField>
        </ReceiptFields>
      </ReceiptSection>
      <ReceiptTable caption="Cuotas" columns={[{ label: 'Cuota' }, { label: 'Monto', numeric: true }]}>
        <ReceiptTableRow>
          <ReceiptTableCell>Cuota 1</ReceiptTableCell>
          <ReceiptTableCell numeric>US$ 500,00</ReceiptTableCell>
        </ReceiptTableRow>
      </ReceiptTable>
      <ReceiptSignatures labels={['Firma del cliente', 'Firma de quien cobra']} />
    </ReceiptDialog>,
  )
}

describe('ReceiptDialog', () => {
  it('shows the branded header, the content and the footer of the receipt', () => {
    renderReceipt()

    const dialog = screen.getByRole('dialog', { name: 'Recibo de prueba' })
    expect(within(dialog).getByText('LoteosAPP')).toBeInTheDocument()
    expect(within(dialog).getByRole('heading', { name: 'Comprobante de prueba' })).toBeInTheDocument()
    expect(within(dialog).getByText('Fecha de emisión')).toBeInTheDocument()
    expect(within(dialog).getByText('07/09/2026')).toBeInTheDocument()
    expect(within(dialog).getByText('Documento generado automáticamente por LoteosAPP.')).toBeInTheDocument()
  })

  it('shows the summary, the fields, the table and the signatures', () => {
    renderReceipt()

    expect(screen.getByText('Total')).toBeInTheDocument()
    expect(screen.getByText('US$ 1.000,00')).toBeInTheDocument()
    expect(screen.getByText('Modalidad')).toBeInTheDocument()
    expect(screen.getByText('Contado')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Datos del comprobante' })).toBeInTheDocument()
    const fields = screen.getByLabelText('Datos')
    expect(within(fields).getByText('Cliente')).toBeInTheDocument()
    expect(within(fields).getByText('Pérez, Ana')).toBeInTheDocument()
    expect(within(fields).getByText('Sin observaciones')).toBeInTheDocument()
    const table = screen.getByRole('table', { name: 'Cuotas' })
    expect(within(table).getByRole('columnheader', { name: 'Monto' })).toBeInTheDocument()
    expect(within(table).getByRole('cell', { name: 'US$ 500,00' })).toBeInTheDocument()
    expect(screen.getByText('Firma del cliente')).toBeInTheDocument()
    expect(screen.getByText('Firma de quien cobra')).toBeInTheDocument()
  })

  it('prints the receipt', async () => {
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    renderReceipt()

    await user.click(screen.getByRole('button', { name: 'Imprimir comprobante' }))

    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()
  })

  it('closes the receipt with the close button and the escape key', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderReceipt({ onClose })

    await user.click(screen.getByRole('button', { name: 'Cerrar' }))
    expect(onClose).toHaveBeenCalledTimes(1)

    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  it('renders nothing while closed', () => {
    renderReceipt({ open: false })

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
