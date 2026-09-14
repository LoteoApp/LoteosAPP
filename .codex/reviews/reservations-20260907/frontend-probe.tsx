import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { Link, MemoryRouter, Route, Routes } from 'react-router'
import ReserveLotDialog from './src/features/reservations/components/ReserveLotDialog'
import ReservationDetailsPage from './src/features/reservations/pages/ReservationDetailsPage'

const mocks=vi.hoisted(()=>({create:vi.fn(),cancel:vi.fn(),get:vi.fn()}))
vi.mock('./src/features/reservations/api/reservations',()=>({createReservation:mocks.create,cancelReservation:mocks.cancel,getReservation:mocks.get}))
vi.mock('./src/features/clients/hooks/use-clients',()=>({useClients:()=>({clientes:[{id:'client',nombre:'Ana',apellido:'Review',dni:'123'}],isLoading:false,error:null})}))
vi.mock('./src/features/reservations/hooks/use-eligible-sellers',()=>({useEligibleSellers:()=>({sellers:[{id:'seller',nombre:'Seller',apellido:'Review',rol:'administrador'}],isLoading:false,error:null})}))
vi.mock('./src/features/auth/hooks/use-auth',()=>({useAuth:()=>({session:{access_token:'token'}})}))
afterEach(()=>vi.clearAllMocks())
const lot={id:'lot-a',manzanaId:'block',numero:'1',estado:'disponible' as const,precio:1000,moneda:'USD' as const,superficie:300,caracteristicas:'',poligono:[]}
const reservation={id:'a',loteoId:'development',loteoNombre:'Development A',loteId:'lot-a',loteNumero:'1',cliente:{id:'client',nombre:'Ana',apellido:'Review',dni:'123'},vendedor:{id:'seller',nombre:'Seller',apellido:'Review',rol:'administrador'},usuarioAlta:{id:'seller',nombre:'Seller',apellido:'Review',rol:'administrador'},estado:'activa',fechaVencimiento:'2026-09-21T12:00:00Z',fechaCreacion:'2026-09-06T12:00:00Z',fechaModificacion:'2026-09-06T12:00:00Z',historial:[]}
it('preserves an uncertain creation key after closing and reopening the dialog',async()=>{
 mocks.create.mockRejectedValue(new TypeError('Failed to fetch'))
 const user=userEvent.setup();render(<ReserveLotDialog accessToken="token" loteoId="development" lote={lot} onCreated={vi.fn()}/>);
 await user.click(screen.getByRole('button',{name:'Reservar lote'}));await user.selectOptions(screen.getByRole('combobox',{name:'Cliente'}),'client');await user.click(screen.getByRole('button',{name:'Confirmar reserva'}));await screen.findByText('Failed to fetch');
 const firstKey=mocks.create.mock.calls[0][2]
 await user.click(screen.getByRole('button',{name:'Cerrar reserva'}));await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
 await user.click(screen.getByRole('button',{name:'Reservar lote'}));await user.selectOptions(screen.getByRole('combobox',{name:'Cliente'}),'client');await user.click(screen.getByRole('button',{name:'Confirmar reserva'}));await waitFor(()=>expect(mocks.create).toHaveBeenCalledTimes(2));
 expect(mocks.create.mock.calls[1][2]).toBe(firstKey)
})
it('does not replace a new reservation detail with an old cancellation response',async()=>{
 let complete:(value:unknown)=>void=()=>{};mocks.cancel.mockImplementation(()=>new Promise(resolve=>{complete=resolve}));mocks.get.mockImplementation((_token,id)=>Promise.resolve(id==='a'?reservation:{...reservation,id:'b',loteoNombre:'Development B'}));
 const user=userEvent.setup();render(<MemoryRouter initialEntries={['/reservas/a']}><Link to="/reservas/b">Open B</Link><Routes><Route path="/reservas/:id" element={<ReservationDetailsPage/>}/></Routes></MemoryRouter>);
 await screen.findByText('Development A');await user.click(screen.getByRole('button',{name:'Cancelar',exact:true}));await user.type(screen.getByRole('textbox',{name:'Justificación de la cancelación'}),'Review reason');await user.click(screen.getByRole('button',{name:'Cancelar reserva',exact:true}));await waitFor(()=>expect(mocks.cancel).toHaveBeenCalledOnce());
 await user.click(screen.getByRole('button',{name:'Cerrar cancelación'}));await user.click(screen.getByRole('link',{name:'Open B'}));await screen.findByText('Development B');
 await act(async()=>{complete({...reservation,estado:'cancelada'})});expect(screen.getByText('Development B')).toBeInTheDocument();expect(screen.queryByText('Development A')).not.toBeInTheDocument()
})
