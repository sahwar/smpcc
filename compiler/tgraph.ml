(* A (directed) graph library, providing transitive closure and topological sort *)

module PSet = BatSet.PSet
module PMap = BatMap.PMap

type 'a graph = ('a, 'a PSet.t) PMap.t

(**********************)
(* Graph constructors *)
(**********************)
let empty = PMap.empty

let get_nodes g =
  PMap.foldi
    (fun node target nodes -> node::nodes) g []

let get_targets g source =
  (* can raise Not_found *)
  PMap.find source g

let add_node g source =
  if PMap.mem source g then g else
  PMap.add source PSet.empty g

let add_edge g source target =
  PMap.add source
    (PSet.add target (PMap.find source (add_node g source)))
    (add_node g target)

let add_edges g source targets =
  PSet.fold
    (fun target g -> add_edge g source target)
    targets
    g

let remove_edge g source target =
  PMap.add source
    (PSet.remove target (PMap.find source (add_node g source)))
    g

let remove_edges g source targets =
  PSet.fold
    (fun target g -> remove_edge g source target)
    g


(*******************)
(* Graph utilities *)
(*******************)

let iter_edges f g =
  PMap.iter (fun source targets ->
    PSet.iter (fun target -> f source target) targets) g

let iter_nodes f g = BatEnum.iter f (PMap.keys g)

(* let fold_edges = PMap.foldi .|. (PSet.fold .|.) *)
let fold_edges f = PMap.foldi (fun n1 -> PSet.fold (f n1))

let get_sources g target =
  fold_edges (fun src tgt srcs -> if tgt = target then PSet.add src srcs else srcs) g PSet.empty

let is_edge g source target =
  PSet.mem target (PMap.find source g)

(* UNDER CONSTRUCTION: *)
(* let make_dot outch g = *)
(*   Printf.fprintf outch "digraph g {\nrankdir=LR\n"; *)
(*   Printf.fprintf outch "node [shape=box,fontname=%S];\n" !font_regular; *)
(*   Printf.fprintf outch "edge [fontname=%S];\n" !font_regular; *)
(*   iter_nodes g in *)
(*   Printf.fprintf outch "}\n" *)

let reverse g =
  fold_edges
    (fun source target g_reversed ->
      add_edge g_reversed target source)
    g
    empty

(**********************************************************************************)
(* Nodes between source and target.  Include source if there is a loop to source. *)
(**********************************************************************************)
let between source target g =
  let visited = ref(PSet.add source (PSet.add target PSet.empty)) in
  let rec label node =
    if PSet.mem node !visited then () else begin
      visited := PSet.add node !visited;
      PSet.iter label (get_targets g node)
    end in
  PSet.iter label (get_targets g source);
  PSet.remove target !visited

(*****************************************************************)
(* Reverse postorder traversal, useful for dominator calculation *)
(*****************************************************************)
let reverse_postorder root g =
  (* start by labeling the nodes of g in postorder, 0..N-1 *)
  let node_to_label = ref PMap.empty in
  let visited = ref PSet.empty in
  let counter = ref 0 in
  let next_count() =
    let count = !counter in
    counter := !counter + 1;
    count in
  let rec label node =
    if PSet.mem node !visited then () else begin
      visited := PSet.add node !visited;
      PSet.iter label (get_targets g node);
      node_to_label := PMap.add node (next_count()) !node_to_label
    end in
  label root;
  (* The reverse postorder sequence is root_label, root_label-1, ..., 0 *)
  let root_label = PMap.find root !node_to_label in
  (* construct a labeled predecessor graph *)
  let labeled_predecessors =
    fold_edges
      (fun source target g -> add_edge g (PMap.find target !node_to_label) (PMap.find source !node_to_label))
      g empty in
  (* for each label record the labels of predecessor nodes *)
  let preds = Array.make (1 + root_label) [] in
  for i = 0 to root_label-1 do (* we don't need the predecessors of the root label *)
    (* PSet.elements is in increasing order, we reverse to get decreasing order *)
    preds.(i) <- List.rev (PSet.elements (PMap.find i labeled_predecessors))
  done;
  !node_to_label, root_label, preds

(******************)
(* Dominator tree *)
(******************)
(* A Simple, Fast Dominance Algorithm, by Keith D. Cooper, Timothy J. Harvey, and Ken Kennedy. Figure 3 *)
let dominator_tree root g =
  let node_to_label, root_label, preds = reverse_postorder root g in
  let label_to_node =
    PMap.foldi (fun n l m -> PMap.add l n m) node_to_label PMap.empty in
  let doms = Array.make (1 + root_label) (-1) in
  doms.(root_label) <- root_label;
  let rec intersect b1 b2 =
    if b1 = b2 then b1 else
    if b1 < b2 then intersect doms.(b1) b2 else
    intersect b1 doms.(b2) in
  let changed = ref true in
  while !changed do
    changed := false;
    for b = root_label-1 downto 0 do
      match preds.(b) with (* labels of predecessors in decreasing order. *)
      | new_idom::other_predecessors -> (* Invariant: new_idom > b, so doms.(new_idom) > 0 *)
          let new_idom = ref new_idom in
          List.iter
            (fun p ->
              if doms.(p) > 0 then new_idom := intersect p !new_idom)
            other_predecessors;
          if doms.(b) != !new_idom then begin
            doms.(b) <- !new_idom;
            changed := true
          end
      | _ -> failwith "impossible: non-root node without a predecessor"
    done
  done;
  BatEnum.fold
    (fun m n -> PMap.add n (PMap.find doms.(PMap.find n node_to_label) label_to_node) m)
    empty
    (PMap.keys g)

(********************)
(* Topological sort *)
(********************)
exception Break
let rec tsort0(input,output,vN,vS,x,k) =
  vS := x::!vS;
  vN := PMap.add x k !vN;
  PSet.iter
    (fun y ->
      if (PMap.find y !vN = 0) then
        tsort0(input,output,vN,vS,y,k+1);
      let vNy = PMap.find y !vN in
      if (vNy < PMap.find x !vN) then
        vN := PMap.add x vNy !vN)
    (get_targets input x);
  if (PMap.find x !vN = k) then begin
    try
      while (!vS <> []) do
        let top = List.hd !vS in
        vN := PMap.add top max_int !vN;
        output := top::!output;
        vS := List.tl !vS;
        if (top = x) then raise Break;
      done
    with Break -> ()
  end

let tsort (input:'a graph) =
  let sources = PMap.foldi
      (fun source targets result -> source::result)
      input [] in
  let output = ref [] in
  let vS = ref [] in
  let vN =
    ref(List.fold_left
          (fun result x -> PMap.add x 0 result)
          PMap.empty
          sources) in
  List.iter
    (fun x ->
      if (PMap.find x !vN = 0) then tsort0(input,output,vN,vS,x,1))
    sources;
  !output


(**********************)
(* Transitive closure *)
(**********************)

(* Based on Esko Nuutila, "An efficient transitive closure algorithm
   for cyclic digraphs," Information Processing Letters 52 (1994)
   207-213.  Modified to allow self-loops. *)

type 'a nodestate = {
    mutable root: 'a;
    mutable c: int;
    mutable visitindex: int;
  }

type 'a componentstate = {
    mutable succ: int PSet.t;
    mutable nodes: 'a PSet.t;
  }

type 'a tcstate = {
    mutable visited: 'a PSet.t;
    mutable vsitindex: int;
    mutable ns: ('a,'a nodestate) PMap.t;
    mutable cs: (int,'a componentstate) PMap.t;
    mutable cindex: int;
    mutable nstack: 'a list;
    mutable cstack: int list;
  }

let union x y = PSet.fold PSet.add x y

let rec comp_tc ts g v =
  ts.visited <- PSet.add v ts.visited;
  let nsv = {root=v; c=0; visitindex=ts.vsitindex} in
  ts.vsitindex <- ts.vsitindex + 1;
  ts.ns <- PMap.add v nsv ts.ns;
  ts.nstack <- v::ts.nstack;
  let hsaved = List.length ts.cstack in (* FIX: inefficient *)
  PSet.iter
    (fun w ->
      if v = w then () else (* self-loop; rest of while would be nop *)
      let ( is_forward_edge, nsw ) =
        if PSet.mem w ts.visited then
          let nsw = PMap.find w ts.ns in
          ( not(nsw.visitindex < nsv.visitindex),
            nsw )
        else begin
          comp_tc ts g w;
          ( false,
            PMap.find w ts.ns )
        end in
      let cw = nsw.c in
      if cw = 0 then begin
        let nsrootv = PMap.find nsv.root ts.ns in
        let nsrootw = PMap.find nsw.root ts.ns in
        (* root(v) = MIN(root(v),root(w)) *)
        if nsrootv.visitindex > nsrootw.visitindex then
          nsv.root <- nsw.root;
      end
      else if not is_forward_edge then
        ts.cstack <- cw::ts.cstack)
    (get_targets g v);
  if nsv.root <> v then () (* return *) else
  let cnew = ts.cindex in
  ts.cindex <- 1 + ts.cindex;
  let csCnew = { succ=PSet.empty; nodes=PSet.empty } in
  ts.cs <- PMap.add cnew csCnew ts.cs;
  (match ts.nstack with
    [] -> ()
  | top::_ ->
      if (top <> v || is_edge g v v) (* check for self-loop *) then
        csCnew.succ <- PSet.add cnew csCnew.succ);
  for hnow = List.length ts.cstack downto hsaved+1 do
    let x = List.hd ts.cstack in
    ts.cstack <- List.tl ts.cstack;
    if not(PSet.mem x csCnew.succ) then
      let s = PSet.add x csCnew.succ in
      let s = union s (PMap.find x ts.cs).succ in
      csCnew.succ <- s
  done;
  let rec loop = function [] -> ()
    | w::tl ->
        ts.nstack <- tl;
        let nsw = PMap.find w ts.ns in
        nsw.c <- cnew;
        csCnew.nodes <- PSet.add w csCnew.nodes;
        if w = v then () else loop ts.nstack
  in
  loop ts.nstack

let tc g =
  let ts =
    { visited=PSet.empty;
      vsitindex=1;
      ns=PMap.empty;
      cs=PMap.empty;
      cindex=1;
      nstack=[];
      cstack=[]; } in
  PMap.iter
    (fun source targets ->
      if not(PSet.mem source ts.visited) then comp_tc ts g source)
    g;
  (* result should have at least all nodes of input *)
  let result = ref
    (List.fold_left add_node empty (get_nodes g)) in
  for c = 1 to ts.cindex - 1 do
    let cs = PMap.find c ts.cs in
    let targets = ref PSet.empty in
    PSet.iter
      (fun c2 ->
        targets := union !targets (PMap.find c2 ts.cs).nodes)
      cs.succ;
    PSet.iter
      (fun v ->
        result := add_edges !result v !targets)
      cs.nodes;
  done;
  !result
