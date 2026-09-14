package main

import (
 "context"
 "database/sql"
 "encoding/json"
 "fmt"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "sort"
 "strings"
 "sync"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgxpool"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pressly/goose/v3"
 "loteosapp/backend/internal/business/domain"
 "loteosapp/backend/internal/business/gateway"
 "loteosapp/backend/internal/infrastructure/repository/postgres"
)
type sample struct { SQL string; Calls int; Milliseconds []float64; Errors []string; Plan json.RawMessage; PlanError string; args []any }
type trace struct { mu sync.Mutex; samples map[string]*sample; enabled bool }
type queryStart struct { at time.Time; sql string; args []any }
type traceKey struct{}
func (t *trace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context { return context.WithValue(ctx,traceKey{},queryStart{time.Now(),data.SQL,append([]any(nil),data.Args...)}) }
func (t *trace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
 if !t.enabled { return }; q:=ctx.Value(traceKey{}).(queryStart)
 t.mu.Lock(); defer t.mu.Unlock()
 s:=t.samples[q.sql]; if s==nil {s=&sample{SQL:q.sql,args:q.args};t.samples[q.sql]=s}
 s.Calls++; s.Milliseconds=append(s.Milliseconds,float64(time.Since(q.at).Microseconds())/1000)
 if data.Err!=nil {s.Errors=append(s.Errors,data.Err.Error())}
}
func must(err error) { if err!=nil {panic(err)} }
func main(){
 ctx,cancel:=context.WithTimeout(context.Background(),15*time.Minute);defer cancel()
 root:=`E:/LoteosAPP`; output:=filepath.Join(root,".codex","reviews","reservations-20260907");must(os.MkdirAll(output,0755))
 admin,err:=pgxpool.New(ctx,os.Getenv("DATABASE_URL"));must(err);defer admin.Close()
 schema:=fmt.Sprintf("review_reservations_%d",time.Now().Unix())
 _,err=admin.Exec(ctx,"CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize());must(err)
 defer func(){cleanupCtx,done:=context.WithTimeout(context.Background(),30*time.Second);defer done();_,err:=admin.Exec(cleanupCtx,"DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE");fmt.Printf("CLEANUP schema=%s error=%v\n",schema,err)}()
 dsn,err:=url.Parse(os.Getenv("DATABASE_URL"));must(err); params:=dsn.Query();params.Set("search_path",schema+",extensions");params.Set("application_name","reservation-review");dsn.RawQuery=params.Encode()
 db,err:=sql.Open("pgx",dsn.String());must(err);db.SetMaxOpenConns(1)
 provider,err:=goose.NewProvider(goose.DialectPostgres,db,os.DirFS(filepath.Join(root,"migrations")));must(err)
 _,err=provider.Up(ctx);must(err);must(db.Close());fmt.Println("ISOLATED MIGRATIONS UP OK",schema)
 tr:=&trace{samples:map[string]*sample{}}
 cfg,err:=pgxpool.ParseConfig(dsn.String());must(err);cfg.MaxConns=4;cfg.ConnConfig.Tracer=tr
 pool,err:=pgxpool.NewWithConfig(ctx,cfg);must(err);defer pool.Close()
 seed(ctx,pool)
 repo:=postgres.NewReservationRepository(pool);tr.enabled=true
 for i:=1;i<=3;i++ {
  cmd:=createCommand(i,time.Now().UTC());start:=time.Now();r,err:=repo.Create(ctx,cmd);fmt.Printf("CREATE run=%d ms=%.2f state=%s error=%v\n",i,time.Since(start).Seconds()*1000,r.Estado,err);must(err)
  _,err=repo.Create(ctx,cmd);must(err);_,err=repo.Get(ctx,r.ID,gateway.ReservationScope{});must(err)
  _,err=repo.ListEligibleSellers(ctx,uuid(1),gateway.ReservationScope{});must(err)
  _,err=repo.Cancel(ctx,gateway.CancelReservationCommand{ReservationID:r.ID,ActorID:uuid(2),Reason:"Review cancellation",CancelledAt:time.Now().UTC()},gateway.ReservationScope{});must(err)
 }
 for _,search:=range []string{"","Review","%"} { start:=time.Now();p,err:=repo.List(ctx,domain.ReservationListFilter{Search:search},gateway.ReservationScope{});fmt.Printf("LIST q=%q ms=%.2f rows=%d total=%d error=%v\n",search,time.Since(start).Seconds()*1000,len(p.Items),p.Total,err) }
 p,err:=repo.List(ctx,domain.ReservationListFilter{Page:9999},gateway.ReservationScope{});fmt.Printf("EMPTY PAGE total=%d pages=%d error=%v\n",p.Total,p.TotalPages,err)
 agencyAuth:=uuid(6);p,err=repo.List(ctx,domain.ReservationListFilter{},gateway.ReservationScope{AssigneeAuthProviderID:&agencyAuth,ByAgencyAssignment:true});fmt.Printf("AGENCY LIST rows=%d total=%d error=%v\n",len(p.Items),p.Total,err)
 _,err=repo.ListEligibleSellers(ctx,uuid(1),gateway.ReservationScope{AssigneeAuthProviderID:&agencyAuth,ByAgencyAssignment:true});must(err)
 old:=createCommand(4,time.Now().UTC().Add(-361*time.Hour));_,err=repo.Create(ctx,old);must(err)
 report,err:=repo.ExpireDue(ctx,time.Now().UTC(),5);fmt.Printf("EXPIRE processed=%d failures=%d error=%v\n",report.Processed,len(report.Failures),err)
 staleClock(ctx,pool,repo);tr.enabled=false;explain(ctx,pool,tr)
 samples:=make([]*sample,0,len(tr.samples));for _,s:=range tr.samples {samples=append(samples,s)};sort.Slice(samples,func(i,j int)bool{return samples[i].SQL<samples[j].SQL})
 b,err:=json.MarshalIndent(samples,"","  ");must(err);must(os.WriteFile(filepath.Join(output,"query-measurements.json"),b,0644));fmt.Println("QUERIES PROFILED",len(samples))
 if os.Getenv("REVIEW_SKIP_TESTS")=="" {
  for _,script:=range []string{"test","test:coverage"} {
   command:=exec.CommandContext(ctx,"cmd.exe","/c","pnpm",script);command.Dir=root;command.Env=append(os.Environ(),"DATABASE_URL="+dsn.String());f,err:=os.Create(filepath.Join(output,strings.ReplaceAll(script,":","-")+".log"));must(err);command.Stdout=f;command.Stderr=f;err=command.Run();f.Close();fmt.Printf("CHECK %s error=%v\n",script,err)
  }
 }
}
func uuid(n int)string{return fmt.Sprintf("90000000-0000-0000-0000-%012d",n)}
func createCommand(n int,now time.Time)gateway.CreateReservationCommand{return gateway.CreateReservationCommand{LoteoID:uuid(1),LoteID:uuid(100+n),ClienteID:uuid(3),VendedorID:uuid(2),ActorID:uuid(2),IdempotencyKey:fmt.Sprintf("review-%d",n),IdempotencyPayloadHash:strings.Repeat("a",64),CreatedAt:now}}
func seed(ctx context.Context,pool *pgxpool.Pool){
 _,err:=pool.Exec(ctx,`
 INSERT INTO usuarios(id,auth_provider_id,email,rol,nombre,apellido,perfil_completo) VALUES ('90000000-0000-0000-0000-000000000002','90000000-0000-0000-0000-000000000007','review@example.invalid','administrador','Review','Admin',true);
 INSERT INTO clientes(id,nombre,apellido,dni) VALUES ('90000000-0000-0000-0000-000000000003','Review','Client','review-fixture');
 INSERT INTO inmobiliarias(id,razon_social) VALUES ('90000000-0000-0000-0000-000000000004','Review agency');
 INSERT INTO usuarios(id,auth_provider_id,email,rol,nombre,apellido,perfil_completo,inmobiliaria_id) VALUES ('90000000-0000-0000-0000-000000000005','90000000-0000-0000-0000-000000000006','agency@example.invalid','inmobiliaria','Review','Agency',true,'90000000-0000-0000-0000-000000000004');
 INSERT INTO loteos(id,nombre) VALUES ('90000000-0000-0000-0000-000000000001','Review development');
 INSERT INTO inmobiliaria_loteos(inmobiliaria_id,loteo_id) VALUES ('90000000-0000-0000-0000-000000000004','90000000-0000-0000-0000-000000000001');
 INSERT INTO manzanas(id,loteo_id,numero) VALUES ('90000000-0000-0000-0000-000000000008','90000000-0000-0000-0000-000000000001','1');
 INSERT INTO lotes(id,loteo_id,manzana_id,numero,precio) SELECT ('90000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'90000000-0000-0000-0000-000000000001','90000000-0000-0000-0000-000000000008',n::text,1000 FROM generate_series(101,5110)n;
 INSERT INTO reservas(lote_id,cliente_id,vendedor_id,usuario_alta,fecha_vencimiento) SELECT id,'90000000-0000-0000-0000-000000000003','90000000-0000-0000-0000-000000000005','90000000-0000-0000-0000-000000000002',now()+interval '360 hours' FROM lotes WHERE numero::int>=111;
 INSERT INTO lote_estados(lote_id,estado,origen,reserva_id,usuario_modificacion) SELECT lote_id,'reservado','reserva',id,usuario_alta FROM reservas;
 ANALYZE usuarios; ANALYZE clientes; ANALYZE inmobiliarias; ANALYZE inmobiliaria_loteos; ANALYZE loteos; ANALYZE lotes; ANALYZE reservas; ANALYZE reserva_estados; ANALYZE lote_estados;
 `);must(err);fmt.Println("FIXTURE 5000 active reservations, 5010 lots")
}
func staleClock(ctx context.Context,pool *pgxpool.Pool,repo *postgres.ReservationRepository){
 due:=time.Now().UTC().Add(6*time.Second);cmd:=createCommand(5,due.Add(-domain.ReservationDuration));r,err:=repo.Create(ctx,cmd);must(err)
 tx,err:=pool.Begin(ctx);must(err);defer tx.Rollback(context.Background());_,err=tx.Exec(ctx,"SELECT id FROM lotes WHERE id=$1::uuid FOR UPDATE",cmd.LoteID);must(err)
 cancelledAt:=time.Now().UTC();ch:=make(chan error,1)
 go func(){r,err:=repo.Cancel(ctx,gateway.CancelReservationCommand{ReservationID:r.ID,ActorID:uuid(2),Reason:"Wait across expiry",CancelledAt:cancelledAt},gateway.ReservationScope{});fmt.Printf("CANCEL ACROSS EXPIRY requested_before_due=%v completed_after_due=%v state=%s error=%v\n",cancelledAt.Before(due),time.Now().After(due),r.Estado,err);ch<-err}()
 delay:=time.Until(due.Add(time.Second));if delay>0{time.Sleep(delay)};must(tx.Commit(ctx));<-ch
}
func explain(ctx context.Context,pool *pgxpool.Pool,tr *trace){
 for _,s:=range tr.samples{
  upper:=strings.ToUpper(strings.TrimSpace(s.SQL));if !strings.HasPrefix(upper,"SELECT")&&!strings.HasPrefix(upper,"INSERT"){continue}
  tx,err:=pool.Begin(ctx);must(err);args:=append([]any(nil),s.args...)
  if strings.HasPrefix(upper,"INSERT INTO RESERVAS") {args[0]=uuid(106);args[6]="explain-create"}
  if strings.HasPrefix(upper,"INSERT INTO RESERVA_ESTADOS") {var id string;must(tx.QueryRow(ctx,"SELECT id::text FROM reservas WHERE estado_actual='activa' LIMIT 1").Scan(&id));args[0]=id}
  if strings.HasPrefix(upper,"INSERT INTO LOTE_ESTADOS") {
   var id string
   if fmt.Sprint(args[1])=="reservado" {must(tx.QueryRow(ctx,"INSERT INTO reservas(lote_id,cliente_id,vendedor_id,usuario_alta,fecha_vencimiento) VALUES($1::uuid,$2::uuid,$3::uuid,$3::uuid,now()+interval '360 hours') RETURNING id::text",uuid(106),uuid(3),uuid(2)).Scan(&id));args[0]=uuid(106);args[5]=&id} else {var lot string;must(tx.QueryRow(ctx,"SELECT id::text,lote_id::text FROM reservas WHERE estado_actual='activa' LIMIT 1").Scan(&id,&lot));args[0]=lot;args[5]=&id}
  }
  var plan []byte;err=tx.QueryRow(ctx,"EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+s.SQL,args...).Scan(&plan)
  if err!=nil{s.PlanError=err.Error()}else{s.Plan=json.RawMessage(plan)};tx.Rollback(context.Background())
 }
}
