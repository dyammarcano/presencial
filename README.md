# Presencial - Controle de Presença Mensal

Este aplicativo em Go permite registrar se você esteve **presencial** em um determinado dia. Ele é ideal para o
controle informal de frequência ou uso recorrente de atividades como reuniões ou medicação.

---

## 🚀 O que o programa faz?

Ao executar, o programa irá:

1. Criar automaticamente uma pasta de dados no diretório do usuário
2. Mostrar uma janela de diálogo perguntando sobre sua presença
3. Exibir o total de presenças quando atingir a meta mensal
4. Salvar o registro em um banco de dados SQLite
5. Minimizar para a bandeja do sistema (tray icon)

---

## 🛠️ Funcionalidades

- Mostra resumo mensal com total de presenças registradas
- Permite configurar uma **meta mensal** (padrão: 4)
- Armazena registros com **data, hora, resposta, observação e área**
- Registro automático ao selecionar a área
- Interface gráfica moderna com Fyne.io
- Ícone na bandeja do sistema para acesso rápido
- Importação e exportação de dados em formato JSON

---

## 📂 Onde os arquivos são salvos?

Os dados são armazenados na pasta:

- **Windows**: `C:\Users\SEU_USUARIO\AppData\Roaming\presencial\`
- **macOS**: `/Users/seu_usuario/Library/Application Support/presencial/`
- **Linux**: `/home/seu_usuario/.local/share/presencial/`

### Arquivos criados:

| Arquivo          | Descrição                                |
|------------------|------------------------------------------|
| `application.db` | Banco de dados SQLite com todos os dados |
| `export_*.json`  | Arquivos de exportação de dados          |

---

## 🧾 Estrutura dos Dados

Os registros contêm as seguintes informações:

- `Date`: Data do registro (YYYY-MM-DD)
- `Time`: Hora do registro (HH:MM:SS)
- `Response`: `"Sim"` ou `"Não"`
- `Observation`: Campo adicional (opcional)
- `Area`: Área escolhida pelo usuário (AG, CT, CEIC, OUTRO)

---

## ⚙️ Configuração

Na primeira execução, o programa solicitará:

- A quantidade de dias presenciais desejada no mês (meta)
- Isso será salvo automaticamente no banco de dados

A meta mensal pode ser alterada a qualquer momento através do menu "Editar > Configurar Meta de Dias".

---

## 🔄 Importação e Exportação

O aplicativo permite:

- **Exportar dados**: Salva todos os registros em um arquivo JSON
- **Importar dados**: Carrega registros de um arquivo JSON previamente exportado

Acesse essas funções através do menu "Arquivo" ou do ícone na bandeja do sistema.

---

## 🖥️ Compatibilidade

Este aplicativo é compatível com:

- ✅ Windows
- ✅ Linux
- ✅ macOS

Requisitos:

- Go 1.23.0+ (para compilação)
- Bibliotecas: Fyne.io, GORM, SQLite

---

## 💡 Exemplo de Dados JSON

```json
[
  {
    "ID": 1,
    "Date": "2025-05-01",
    "Time": "08:45:12",
    "Response": "Sim",
    "Observation": "",
    "Area": "CT"
  },
  {
    "ID": 2,
    "Date": "2025-05-02",
    "Time": "08:47:30",
    "Response": "Sim",
    "Observation": "extra",
    "Area": "CEIC"
  }
]
