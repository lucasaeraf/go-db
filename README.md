# go-db
Personal project to practice Go programming language and database internals knowledge
## Concepts
---
- **Atomicity and durability**
- **KV Store based on B-tree**
- **Relational DB on top of KV**
- **Concurrency Control**

## Chapter 1
---
### Salvando dados em um arquivo
---
Para salvar dados em algum arquivo temos algumas opções:
1. Colocar os dados novos no arquivo e syncar no disco;
2. Colocar os dados novos em um novo arquivo, syncar no disco e renomear o novo arquivo igual o arquivo antigo;
Opção 2 é mais segura pra evitar que pessoas estejam lendo coisas diferentes, mas ao mesmo tempo é necessário uma autoridade central para controlar quem está escrevendo para evitar concorrência nessa escrita.

### Usar logs para persistência e recuperação de dados
---
Vamos utilizar a estratégia de ter arquivos de logs **APPEND-ONLY** com todas as instruções que são executadas no banco de dados e iremos, da melhor forma possível, manter o ordenamento dessas entradas de instruções juntamente com um checksum para ter a certeza de que, com a execução sequencial das instruções cheguemos ao estado correto do nosso banco de dados. Porém isso não é uma solução para indexação, ou seja, quando vamos buscar um dado.

## Chapter 2 - Estruturas de Dados para Indexação
---
### Tipos de queries
---
Como tudo em computação, temos vários _trade-offs_ para levar em consideração ao escolher as estruturas de dados pro projeto, e, mais do que isso, _there is no silver bullet_. Leva-se em consideração então os tipos de queries que são feitos para o DB:
1. Scan the whole data set. (No index is used).
2. Point query: Query the index by a specific key.
3. Range query: Query the index by a range. (The index is sorted).

### B-trees
---
Essa é a escolha para a estrutura de dados principal. Por quê? Para reduzir o número de lookups no disco que é o maior fator de latência. B-trees são árvores n-árias e, para nossa aplicação, cada nível da árvore representa uma página no disco, ou seja, quanto mais fundo temos que andar na árvore, maior o número de lookups no disco que temos que fazer (e gastar mais memória).

Do chatgpt:
#### **B-Trees:**

- A **B-Tree** is a generalization of a binary search tree, where nodes can have more than two children.
- **Balanced Tree:** All leaf nodes are at the same depth, and the tree remains balanced after insertions and deletions.
- **Node Structure:** A B-Tree node contains multiple keys and child pointers. Each key in the node acts as a separator for the child nodes, and child nodes store keys in sorted order.
- **Order of a B-Tree (m):** The maximum number of children a node can have is `m`. A B-Tree of order `m` must have between `ceil(m/2)` and `m` children per node (except the root).
- **Search Operation:** B-Tree performs a binary search within a node, and then based on the result, traverses to the appropriate child node, and repeats until it reaches a leaf.

#### **B+Trees:**

- **B+Tree** is a variant of the B-Tree, but with a few differences:
    - All keys are stored in the leaf nodes, and the internal nodes only store pointers to guide the search path.
    - **Linked Leaves:** Leaf nodes in B+Tree are linked to each other, allowing efficient range queries and in-order traversal.
    - Internal nodes do not store actual data (keys and values), only routing information.
- **Advantages of B+Tree:**
	- Fast range queries: Since the leaf nodes are linked, it supports fast traversal of sequential data.
	- More compact internal nodes: Internal nodes only contain pointers to child nodes, leading to better cache utilization.

### LSM-Trees
---
Essas são estruturas de dados para armazenamento em log



## Chapter 3 - B+Trees e Crash Recovery
---
### B+Trees em disco
---
A gente precisa levar em consideração a alocação de espaço em disco para persistência dos dados na memória secundária. Nesse momento o que tem que ser levado em conta é o tipo de file system que tá sendo usado e como ele faz alocação páginas no disco. Vamos considerar aqui que todas os nós de uma B+Tree são uma página de mesmo tamanho - isso é feito para facilitar o processo de liberar espaço em disco para reuso a posteriori.

A estratégia para não perder dados e obter o caráter de **Crash Recovery** é chamada de [[copy-on-write]] 

## Chapter 4 - B+Tree Node and Insertion
---
### Simplificações
---
Nesse ponto começamos a implementação das estruturas em si, começando pelos B+Tree Nodes. Cada nó vai possuir os seguintes atributos:

```
## Nodes
| type | nkeys | pointers   | offsets    | key-values | unused |
| 2B   | 2B    | nkeys * 8B | nkeys * 2B |     ...    |        |

## Key values
| klen | vlen | key | val |
| 2B   | 2B   | ... | ... |

```

Isso inclui algumas simplificações como o fato de que poderíamos diferencias os nós folha e os nós internos já que os nós folha não possuem ponteiros e, por sua vez, os nós internos nao possuem valores