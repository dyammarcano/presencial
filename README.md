
O programa irá:
1. Criar automaticamente uma pasta "Precencial" no diretório do usuário
2. Mostrar uma janela de diálogo perguntando sobre sua presença
3. Exibir o total de presenças quando atingir a meta mensal
4. Salvar o registro em um arquivo CSV

## Estrutura dos Dados

Os registros são salvos em `~/Precencial/registros.csv` com as seguintes colunas:
- data: Data do registro (YYYY-MM-DD)
- hora: Hora do registro (HH:MM:SS)
- resposta: "Sim" ou "Não"
- observacao: Campo adicional para registros extras

## Configuração

A meta mensal está definida como 8 presenças e pode ser ajustada modificando a variável `meta_mensal` no código.

## Suporte

O sistema é compatível com:
- Windows
- Linux
- macOS

## Contribuição

Para contribuir com o projeto:
1. Faça um fork do repositório
2. Crie uma branch para sua feature
3. Envie suas alterações com commits descritivos
4. Abra um Pull Request
