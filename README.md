# Precencial - Controle de Presença Mensal

Este aplicativo em Python permite registrar se você esteve **presencial** em um determinado dia. Ele é ideal para o
controle informal de frequência ou uso recorrente de atividades como reuniões ou medicação.

---

## 🚀 O que o programa faz?

Ao executar, o programa irá:

1. Criar automaticamente uma pasta `Precencial` no diretório do usuário
2. Mostrar uma janela de diálogo perguntando sobre sua presença
3. Exibir o total de presenças quando atingir a meta mensal
4. Salvar o registro em um arquivo CSV

---

## 🛠️ Funcionalidades

- Mostra resumo mensal com total de presenças registradas
- Permite configurar uma **meta mensal** (padrão: 8)
- Armazena registros com **data, hora, resposta, observação e área**
- Registro automático ao selecionar a área
- Interface com fontes monoespaçadas para melhor legibilidade
- Logs salvos localmente

---

## 📂 Onde os arquivos são salvos?

Os dados são armazenados na pasta:

- **Windows**: `C:\Usuários\SEU_USUARIO\Precencial\`
- **Linux/macOS**: `/home/seu_usuario/Precencial/`

### Arquivos criados:

| Arquivo                | Descrição                          |
|------------------------|------------------------------------|
| `registros.csv`        | Armazena todos os registros        |
| `config.txt`           | Meta mensal definida pelo usuário  |
| `presence_tracker.log` | Log de execução e erros do sistema |

---

## 🧾 Estrutura dos Dados

Os registros são salvos em `~/Precencial/registros.csv` com as colunas:

- `data`: Data do registro (YYYY-MM-DD)
- `hora`: Hora do registro (HH:MM:SS)
- `resposta`: `"Sim"` ou `"Não"`
- `observacao`: Campo adicional (ex: `"extra"`)
- `area`: Área escolhida pelo usuário (AG, CT, CEIC, OUTRO)

---

## ⚙️ Configuração

Na primeira execução, o programa solicitará:

- A quantidade de dias presenciais desejada no mês (meta)
- Isso será salvo automaticamente em `config.txt`

A meta mensal pode ser alterada manualmente ou reconfigurada deletando o arquivo `config.txt`.

---

## 🖥️ Compatibilidade

Este aplicativo é compatível com:

- ✅ Windows
- ✅ Linux
- ✅ macOS

Requisitos:

- Python 3.7+
- Módulos padrão: `tkinter`, `csv`, `enum`, `dataclasses`, `logging`

---

## 💡 Exemplo de Registro

```csv
data,hora,resposta,observacao,area
2025-05-01,08:45:12,Sim,,CT
2025-05-02,08:47:30,Sim,extra,CEIC
2025-05-03,08:50:03,Não,,N/A
