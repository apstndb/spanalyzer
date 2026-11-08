from pathlib import Path
import hashlib,json,struct,subprocess,shutil
s=Path('<LOCAL_CHECKOUT>/.tmp/gql-order-root-cause-20261030');out=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/evidence');data=(s/'emulator_main.original').read_bytes();assert data[:6]==b'\x7fELF\x02\x01';phoff=struct.unpack_from('<Q',data,32)[0];ents,num=struct.unpack_from('<HH',data,54)
def fileoff(addr):
 for i in range(num):
  typ,flags,off,va,pa,fs,ms,align=struct.unpack_from('<IIQQQQQQ',data,phoff+i*ents)
  if typ==1 and va<=addr<va+fs:return off+addr-va
 raise ValueError('address absent')
variants=[('original-copy',None,None,None,'Identical binary copied into the same base image.'),('no-scramble',0x682d74,0x39008261,0x3900827f,'CommonEvaluatorOptions: strb w1,[x19,#0x20] -> strb wzr,[x19,#0x20]; only scramble_undefined_orderings is false.'),('graph-ordered',0x94dd38,0x39412265,0x52800025,'AlgebrizeGraphTableScan: load graph_table_scan->is_ordered into w5 -> mov w5,#1 before AlgebrizeProjectScanInternal; global scrambling remains true. Diagnostic intervention, not a reviewed general fix.')];proof=[]
for name,addr,before,after,meaning in variants:
 b=bytearray(data);off=None
 if addr is not None:
  off=fileoff(addr);assert struct.unpack_from('<I',b,off)[0]==before;(b[off:off+4])=struct.pack('<I',after)
 changes=[i for i,(x,y)in enumerate(zip(data,b))if x!=y];d=s/'images'/name;d.mkdir(parents=True,exist_ok=True);target=d/'emulator_main';target.write_bytes(b);target.chmod(0o755);(d/'Dockerfile').write_text('FROM gcr.io/cloud-spanner-emulator/emulator:1.5.59@sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8\nCOPY --chmod=0755 emulator_main /emulator_main\n');proof.append({'variant':name,'original_binary_sha256':hashlib.sha256(data).hexdigest(),'variant_binary_sha256':hashlib.sha256(b).hexdigest(),'size':len(b),'instruction_address':hex(addr)if addr else None,'file_offset':off,'before_instruction':hex(before)if before else None,'after_instruction':hex(after)if after else None,'changed_byte_offsets':changes,'meaning':meaning})
 if addr:
  r=subprocess.run(['/opt/homebrew/opt/llvm/bin/llvm-objdump','--disassemble','--start-address='+hex(addr-8),'--stop-address='+hex(addr+12),str(target)],capture_output=True,text=True);assert r.returncode==0;(out/(name+'-patched.disassembly.txt')).write_text(r.stdout)
(out/'binary-interventions.json').write_text(json.dumps(proof,indent=2)+'\n');print(json.dumps(proof))
