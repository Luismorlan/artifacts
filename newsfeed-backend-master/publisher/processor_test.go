package publisher

import (
	b64 "encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/rnr-capital/newsfeed-backend/deduplicator"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/server/graph/generated"
	"github.com/rnr-capital/newsfeed-backend/server/resolver"
	. "github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
)

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

type TestMessageQueueReader struct {
	msgs []*MessageQueueMessage
}

func (reader *TestMessageQueueReader) DeleteMessage(msg *MessageQueueMessage) error {
	return nil
}

// Always return all messages
func (reader *TestMessageQueueReader) ReceiveMessages(maxNumberOfMessages int64) (msgs []*MessageQueueMessage, err error) {
	return reader.msgs, nil
}

// Pass in all the crawler messages that will be used for testing
// Reader will return queue message object
func NewTestMessageQueueReader(crawlerMsgs []*protocol.CrawlerMessage) *TestMessageQueueReader {
	var res TestMessageQueueReader
	var queueMsgs []*MessageQueueMessage

	for _, m := range crawlerMsgs {
		encodedBytes, _ := proto.Marshal(m)
		str := b64.StdEncoding.EncodeToString(encodedBytes)
		var msg MessageQueueMessage
		msg.Message = &str
		queueMsgs = append(queueMsgs, &msg)
	}
	res.msgs = queueMsgs
	return &res
}

func TestCutToToken(t *testing.T) {
	content := "LLM"
	assert.Equal(t, cutToTokens(content, 7000), "LLM")
}

func TestCalculateEmbedding(t *testing.T) {
	db, err := GetTestingDBConnection()
	assert.Nil(t, err)
	origin := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				Id: "2",
			},
			Content:            "LLMs 入门实战系列向AI转型的程序员都关注了这个号👇👇👇【LLMs 入门实战系列】第五重 GPT4ALL第十一重 OpenBuddy第十二重 Baize第十三重 OpenChineseLLaMA第十四重 Panda第十五重 Ziya-LLaMA-13B第十六重 BiLLa第十七重 Luotuo-Chinese-LLM第十八重 Linly第十九重 ChatYuan第二十重 CPM-Bee第二十一重 TigerBot第二十二重 书生·浦语第二十三重 Aquila第一重 金融领域第二重 医疗领域第三重 法律领域第四重 教育领域第五重 文化领域第六重 Coding第一重 AutoGPT第二重 Knowledge Extraction第一重 Massively Multilingual Speech (MMS，大规模多语种语音)第二重 Retrieval-based-Voice-Conversion第一重 Massively Multilingual Speech (MMS，大规模多语种语音)第二重 whisper第一重 BLIP第二重 BLIP2第三重 MiniGPT-4第四重 VisualGLM-6B第五重 Ziya-Visual第一重 Stable Diffusion第一重 langchain第二重 wenda第一重 分布式训练神器第二重 LLMs Trick第三重 LLMTune第四重 QLoRA第一重 ChatGLM-6B 系列第十一重 Llama2第十重 Baichuan第二重 Stanford Alpaca 7B第三重 Chinese-LLaMA-Alpaca第四重 小羊驼 Vicuna第五重 MOSS第六重 BLOOMz第七重 BELLE第八重 ChatRWKV第九重 ChatGPTChatGLM-6BChatGLM2-6BBaichuan-13Bbaichuan-7B第一层 LLMs to Natural Language Processing (NLP)第二层 LLMs to Parameter Efficient Fine-Tuning (PEFT)第三层 LLMs to Intelligent Retrieval (IR)第四层 LLMs to Text-to-Image第五层 LLMs to Visual Question Answering (VQA)第六层 LLMs to Automatic Speech Recognition (ASR)第七层 LLMs to Text To Speech (TTS)第八层 LLMs to Artifact第九层 LLMs to Vertical Field (VF)LLaMA 衍生物系列第一层 LLMs to Natural Language Processing (NLP)第一重 ChatGLM-6B 系列ChatGLM-6B【ChatGLM-6B入门-一】清华大学开源中文版ChatGLM-6B模型学习与实战介绍：ChatGLM-6B 环境配置 和 部署【ChatGLM-6B入门-二】清华大学开源中文版ChatGLM-6B模型微调实战ChatGLM-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.【ChatGLM-6B入门-三】ChatGLM 特定任务微调实战【ChatGLM-6B入门-四】ChatGLM + LoRA 进行finetune介绍：ChatGLM-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.ChatGLM-6B 小编填坑记介绍：ChatGLM-6B 在 部署和微调 过程中 会遇到很多坑，小编掉坑了很多次，为防止 后人和小编一样继续掉坑，小编索性把遇到的坑都填了。【LLMs学习】关于大模型实践的一些总结【LLMs 入门实战 —— 十一 】基于 🤗PEFT 的高效 🤖ChatGLM-6B 微调ChatGLM-6B Freeze 微调：Fine-tuning the MLPs in the last n blocks of the model.ChatGLM-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.ChatGLM-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.微调方式：【LLMs 入门实战】基于 🤗QLoRA 的高效 🤖ChatGLM-6B 微调介绍：本项目使用 https://github.com/huggingface/peft 库，实现了 ChatGLM-6B 模型4bit的 QLoRA 高效微调，可以在一张RTX3060上完成全部微调过程。【LLMs 入门实战 】🤖ChatGLM-6B 模型结构代码解析介绍：ChatGLM-6B 模型结构代码解析ChatGLM2-6B【ChatGLM2-6B入门】清华大学开源中文版ChatGLM-6B模型学习与实战更强大的性能：基于 ChatGLM 初代模型的开发经验，我们全面升级了 ChatGLM2-6B 的基座模型。ChatGLM2-6B 使用了 GLM 的混合目标函数，经过了 1.4T 中英标识符的预训练与人类偏好对齐训练，评测结果显示，相比于初代模型，ChatGLM2-6B 在 MMLU（+23%）、CEval（+33%）、GSM8K（+571%） 、BBH（+60%）等数据集上的性能取得了大幅度的提升，在同尺寸开源模型中具有较强的竞争力。更长的上下文：基于 FlashAttention 技术，我们将基座模型的上下文长度（Context Length）由 ChatGLM-6B 的 2K 扩展到了 32K，并在对话阶段使用 8K 的上下文长度训练，允许更多轮次的对话。但当前版本的 ChatGLM2-6B 对单轮超长文档的理解能力有限，我们会在后续迭代升级中着重进行优化。更高效的推理：基于 Multi-Query Attention 技术，ChatGLM2-6B 有更高效的推理速度和更低的显存占用：在官方的模型实现下，推理速度相比初代提升了 42%，INT4 量化下，6G 显存支持的对话长度由 1K 提升到了 8K。更开放的协议：ChatGLM2-6B 权重对学术研究完全开放，在获得官方的书面许可后，亦允许商业使用。如果您发现我们的开源模型对您的业务有用，我们欢迎您对下一代模型 ChatGLM3 研发的捐赠。论文名称：ChatGLM2-6B: An Open Bilingual Chat LLM | 开源双语对话语言模型论文地址：Github 代码：https://github.com/THUDM/ChatGLM2-6B动机：在主要评估LLM模型中文能力的 C-Eval 榜单中，截至6月25日 ChatGLM2 模型以 71.1 的分数位居 Rank 0 ，ChatGLM2-6B 模型以 51.7 的分数位居 Rank 6，是榜单上排名最高的开源模型。介绍：ChatGLM2-6B 是开源中英双语对话模型 ChatGLM-6B 的第二代版本，在保留了初代模型对话流畅、部署门槛较低等众多优秀特性的基础之上，ChatGLM2-6B 引入了如下新特性：【关于 ChatGLM2 + LoRA 进行finetune 】那些你不知道的事论文名称：ChatGLM2-6B: An Open Bilingual Chat LLM | 开源双语对话语言模型论文地址：Github 代码：https://github.com/THUDM/ChatGLM2-6B介绍：本教程主要介绍对于 ChatGLM2-6B 模型基于 LoRA 进行finetune。【LLMs 入门实战 】基于 🤗PEFT 的高效 🤖ChatGLM2-6B 微调ChatGLM2-6B Freeze 微调：Fine-tuning the MLPs in the last n blocks of the model.ChatGLM2-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.ChatGLM2-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.微调方式：【LLMs 入门实战】基于 🤗QLoRA 的高效 🤖ChatGLM2-6B 微调介绍：本项目使用 https://github.com/huggingface/peft 库，实现了 ChatGLM2-6B 模型4bit的 QLoRA 高效微调，可以在一张RTX3060上完成全部微调过程。第十一重 Llama2【LLMs 入门实战】 Llama2 模型学习与实战官网：https://ai.meta.com/llama/论文名称：《Llama 2: Open Foundation and Fine-Tuned Chat Models》论文地址：https://ai.meta.com/research/publications/llama-2-open-foundation-and-fine-tuned-chat-models/演示平台：https://llama2.ai/Github 代码：https://github.com/facebookresearch/llama模型下载地址：https://ai.meta.com/resources/models-and-libraries/llama-downloads/介绍：此次 Meta 发布的 Llama 2 模型系列包含 70 亿、130 亿和 700 亿三种参数变体。此外还训练了 340 亿参数变体，但并没有发布，只在技术报告中提到了。据介绍，相比于 Llama 1，Llama 2 的训练数据多了 40%，上下文长度也翻倍，并采用了分组查询注意力机制。具体来说，Llama 2 预训练模型是在 2 万亿的 token 上训练的，精调 Chat 模型是在 100 万人类标记数据上训练的。【LLMs 入门实战】Chinese-Llama-2-7b 模型学习与实战https://huggingface.co/ziqingyang/chinese-llama-2-7bhttps://huggingface.co/LinkSoul/Chinese-Llama-2-7b-4bit官网：https://ai.meta.com/llama/论文名称：《Llama 2: Open Foundation and Fine-Tuned Chat Models》论文地址：https://ai.meta.com/research/publications/llama-2-open-foundation-and-fine-tuned-chat-models/演示平台：https://huggingface.co/spaces/LinkSoul/Chinese-Llama-2-7bGithub 代码：https://github.com/LinkSoul-AI/Chinese-Llama-2-7b模型下载地址：介绍：自打 LLama-2 发布后就一直在等大佬们发布 LLama-2 的适配中文版，也是这几天蹲到了一版由 LinkSoul 发布的 Chinese-Llama-2-7b，其共发布了一个常规版本和一个 4-bit 的量化版本，今天我们主要体验下 Llama-2 的中文逻辑顺便看下其训练样本的样式，后续有机会把训练和微调跑起来。第十重 BaichuanBaichuan-13B【LLMs 入门实战 】 Baichuan-13B 模型学习与实战更大尺寸、更多数据：Baichuan-13B 在 Baichuan-7B 的基础上进一步扩大参数量到 130 亿，并且在高质量的语料上训练了 1.4 万亿 tokens，超过 LLaMA-13B 40%，是当前开源 13B 尺寸下训练数据量最多的模型。支持中英双语，使用 ALiBi 位置编码，上下文窗口长度为 4096。同时开源预训练和对齐模型：预训练模型是适用开发者的“基座”，而广大普通用户对有对话功能的对齐模型具有更强的需求。因此本次开源同时发布了对齐模型（Baichuan-13B-Chat），具有很强的对话能力，开箱即用，几行代码即可简单的部署。更高效的推理：为了支持更广大用户的使用，本次同时开源了 int8 和 int4 的量化版本，相对非量化版本在几乎没有效果损失的情况下大大降低了部署的机器资源门槛，可以部署在如 Nvidia 3090 这样的消费级显卡上。开源免费可商用：Baichuan-13B 不仅对学术研究完全开放，开发者也仅需邮件申请并获得官方商用许可后，即可以免费商用。官方微调过（指令对齐）:https://huggingface.co/baichuan-inc/Baichuan-13B-Chat预训练大模型（未经过微调）:https://huggingface.co/baichuan-inc/Baichuan-13B-Basebaichuan-inc/Baichuan-13B：https://github.com/baichuan-inc/Baichuan-13BBaichuan-13B 大模型：介绍：Baichuan-13B 是由百川智能继 Baichuan-7B 之后开发的包含 130 亿参数的开源可商用的大规模语言模型，在权威的中文和英文 benchmark 上均取得同尺寸最好的效果。Baichuan-13B 有如下几个特点：baichuan-7B【LLMs 入门实战 】 baichuan-7B 学习与实战论文名称：论文地址：Github 代码： https://github.com/baichuan-inc/baichuan-7B模型：介绍：由百川智能开发的一个开源可商用的大规模预训练语言模型。基于Transformer结构，在大约1.2万亿tokens上训练的70亿参数模型，支持中英双语，上下文窗口长度为4096。在标准的中文和英文权威benchmark（C-EVAL/MMLU）上均取得同尺寸最好的效果。第二重 Stanford Alpaca 7B【LLMs 入门实战 —— 五 】Stanford Alpaca 7B 模型学习与实战介绍：本教程提供了对LLaMA模型进行微调的廉价亲民 LLMs 学习和微调 方式，主要介绍对于 Stanford Alpaca 7B 模型在特定任务上 的 微调实验，所用的数据为OpenAI提供的GPT模型API生成质量较高的指令数据（仅52k）。第三重 Chinese-LLaMA-Alpaca【LLMs 入门实战 —— 六 】Chinese-LLaMA-Alpaca 模型学习与实战介绍：本教程主要介绍了 Chinese-ChatLLaMA,提供中文对话模型 ChatLLama 、中文基础模型 LLaMA-zh 及其训练数据。模型基于 TencentPretrain 多模态预训练框架构建第四重 小羊驼 Vicuna【LLMs 入门实战 —— 七 】小羊驼 Vicuna模型学习与实战介绍：UC伯克利学者联手CMU、斯坦福等，再次推出一个全新模型70亿/130亿参数的Vicuna，俗称「小羊驼」（骆马）。小羊驼号称能达到GPT-4的90%性能第五重 MOSS【LLMs 入门实战 —— 十三 】MOSS 模型学习与实战介绍：MOSS是一个支持中英双语和多种插件的开源对话语言模型，moss-moon系列模型具有160亿参数，在FP16精度下可在单张A100/A800或两张3090显卡运行，在INT4/8精度下可在单张3090显卡运行。MOSS基座语言模型在约七千亿中英文以及代码单词上预训练得到，后续经过对话指令微调、插件增强学习和人类偏好训练具备多轮对话能力及使用多种插件的能力。局限性：由于模型参数量较小和自回归生成范式，MOSS仍然可能生成包含事实性错误的误导性回复或包含偏见/歧视的有害内容，请谨慎鉴别和使用MOSS生成的内容，请勿将MOSS生成的有害内容传播至互联网。若产生不良后果，由传播者自负。第六重 BLOOMz【LLMs 入门实战 —— 十四 】 BLOOMz 模型学习与实战介绍：大型语言模型（LLMs）已被证明能够根据一些演示或自然语言指令执行新的任务。虽然这些能力已经导致了广泛的采用，但大多数LLM是由资源丰富的组织开发的，而且经常不对公众开放。作为使这一强大技术民主化的一步，我们提出了BLOOM，一个176B参数的开放性语言模型，它的设计和建立要感谢数百名研究人员的合作。BLOOM是一个仅有解码器的Transformer语言模型，它是在ROOTS语料库上训练出来的，该数据集包括46种自然语言和13种编程语言（共59种）的数百个来源。我们发现，BLOOM在各种基准上取得了有竞争力的性能，在经历了多任务提示的微调后，其结果更加强大。模型地址：https://huggingface.co/bigscience/bloomz第七重 BELLE【LLMs 入门实战 —— 十五 】 BELLE 模型学习与实战介绍：相比如何做好大语言模型的预训练，BELLE更关注如何在开源预训练大语言模型的基础上，帮助每一个人都能够得到一个属于自己的、效果尽可能好的具有指令表现能力的语言模型，降低大语言模型、特别是中文大语言模型的研究和应用门槛。为此，BELLE项目会持续开放指令训练数据、相关模型、训练代码、应用场景等，也会持续评估不同训练数据、训练算法等对模型表现的影响。BELLE针对中文做了优化，模型调优仅使用由ChatGPT生产的数据（不包含任何其他数据）。github 地址: https://github.com/LianjiaTech/BELLE第八重 ChatRWKV【LLMs 入门实战 —— 十八 】 ChatRWKV 模型学习与实战Raven 模型：适合直接聊天，适合 +i 指令。有很多种语言的版本，看清楚用哪个。适合聊天、完成任务、写代码。可以作为任务去写文稿、大纲、故事、诗歌等等，但文笔不如 testNovel 系列模型。Novel-ChnEng 模型：中英文小说模型，可以用 +gen 生成世界设定（如果会写 prompt，可以控制下文剧情和人物），可以写科幻奇幻。不适合聊天，不适合 +i 指令。Novel-Chn 模型：纯中文网文模型，只能用 +gen 续写网文（不能生成世界设定等等），但是写网文写得更好（也更小白文，适合写男频女频）。不适合聊天，不适合 +i 指令。Novel-ChnEng-ChnPro 模型：将 Novel-ChnEng 在高质量作品微调（名著，科幻，奇幻，古典，翻译，等等）。目前 RWKV 有大量模型，对应各种场景，各种语言，请选择合适的模型：github: https://github.com/BlinkDL/ChatRWKV模型文件：https://huggingface.co/BlinkDL第九重 ChatGPT《ChatGPT Prompt Engineering for Developers》 学习 之 如何 编写 Prompt？第一个方面：编写清晰、具体的指令第二个方面：给模型些时间思考吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》动机：吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》课程介绍：如何编写 Prompt:《ChatGPT Prompt Engineering for Developers》 学习 之 如何 优化 Prompt？吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》动机：吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》课程介绍：优化编写好 Prompt《ChatGPT Prompt Engineering for Developers》 学习 之 如何使用 Prompt 处理 NLP特定任务？吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》动机：吴恩达老师与OpenAI合作推出《ChatGPT Prompt Engineering for Developers》课程介绍：如何构建ChatGPT Prompt以处理文本摘要、推断和转换(翻译、纠错、风格转换、格式转换等)这些常见的NLP任务第二层 LLMs to Parameter Efficient Fine-Tuning (PEFT)第一重 分布式训练神器分布式训练神器 之 ZeRO 学习动机：虽然 DataParallel (DP) 因为简单易实现，所以目前应用相比于其他两种 广泛，但是 由于 DataParallel (DP) 需要 每张卡都存储一个模型，导致 显存大小 成为 制约模型规模 的 主要因素。核心思路：去除数据并行中的冗余参数，使每一个人都爱我哈哈哈, LLMs 入门实战系列向AI转型的程序员都关注了这个号👇👇👇【LLMs 入门实战系列】第五重 GPT4ALL第十一重 OpenBuddy第十二重 Baize第十三重 OpenChineseLLaMA第十四重 Panda第十五重 Ziya-LLaMA-13B第十六重 BiLLa第十七重 Luotuo-Chinese-LLM第十八重 Linly第十九重 ChatYuan第二十重 CPM-Bee第二十一重 TigerBot第二十二重 书生·浦语第二十三重 Aquila第一重 金融领域第二重 医疗领域第三重 法律领域第四重 教育领域第五重 文化领域第六重 Coding第一重 AutoGPT第二重 Knowledge Extraction第一重 Massively Multilingual Speech (MMS，大规模多语种语音)第二重 Retrieval-based-Voice-Conversion第一重 Massively Multilingual Speech (MMS，大规模多语种语音)第二重 whisper第一重 BLIP第二重 BLIP2第三重 MiniGPT-4第四重 VisualGLM-6B第五重 Ziya-Visual第一重 Stable Diffusion第一重 langchain第二重 wenda第一重 分布式训练神器第二重 LLMs Trick第三重 LLMTune第四重 QLoRA第一重 ChatGLM-6B 系列第十一重 Llama2第十重 Baichuan第二重 Stanford Alpaca 7B第三重 Chinese-LLaMA-Alpaca第四重 小羊驼 Vicuna第五重 MOSS第六重 BLOOMz第七重 BELLE第八重 ChatRWKV第九重 ChatGPTChatGLM-6BChatGLM2-6BBaichuan-13Bbaichuan-7B第一层 LLMs to Natural Language Processing (NLP)第二层 LLMs to Parameter Efficient Fine-Tuning (PEFT)第三层 LLMs to Intelligent Retrieval (IR)第四层 LLMs to Text-to-Image第五层 LLMs to Visual Question Answering (VQA)第六层 LLMs to Automatic Speech Recognition (ASR)第七层 LLMs to Text To Speech (TTS)第八层 LLMs to Artifact第九层 LLMs to Vertical Field (VF)LLaMA 衍生物系列第一层 LLMs to Natural Language Processing (NLP)第一重 ChatGLM-6B 系列ChatGLM-6B【ChatGLM-6B入门-一】清华大学开源中文版ChatGLM-6B模型学习与实战介绍：ChatGLM-6B 环境配置 和 部署【ChatGLM-6B入门-二】清华大学开源中文版ChatGLM-6B模型微调实战ChatGLM-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.【ChatGLM-6B入门-三】ChatGLM 特定任务微调实战【ChatGLM-6B入门-四】ChatGLM + LoRA 进行finetune介绍：ChatGLM-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.ChatGLM-6B 小编填坑记介绍：ChatGLM-6B 在 部署和微调 过程中 会遇到很多坑，小编掉坑了很多次，为防止 后人和小编一样继续掉坑，小编索性把遇到的坑都填了。【LLMs学习】关于大模型实践的一些总结【LLMs 入门实战 —— 十一 】基于 🤗PEFT 的高效 🤖ChatGLM-6B 微调ChatGLM-6B Freeze 微调：Fine-tuning the MLPs in the last n blocks of the model.ChatGLM-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.ChatGLM-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.微调方式：【LLMs 入门实战】基于 🤗QLoRA 的高效 🤖ChatGLM-6B 微调介绍：本项目使用 https://github.com/huggingface/peft 库，实现了 ChatGLM-6B 模型4bit的 QLoRA 高效微调，可以在一张RTX3060上完成全部微调过程。【LLMs 入门实战 】🤖ChatGLM-6B 模型结构代码解析介绍：ChatGLM-6B 模型结构代码解析ChatGLM2-6B【ChatGLM2-6B入门】清华大学开源中文版ChatGLM-6B模型学习与实战更强大的性能：基于 ChatGLM 初代模型的开发经验，我们全面升级了 ChatGLM2-6B 的基座模型。ChatGLM2-6B 使用了 GLM 的混合目标函数，经过了 1.4T 中英标识符的预训练与人类偏好对齐训练，评测结果显示，相比于初代模型，ChatGLM2-6B 在 MMLU（+23%）、CEval（+33%）、GSM8K（+571%） 、BBH（+60%）等数据集上的性能取得了大幅度的提升，在同尺寸开源模型中具有较强的竞争力。更长的上下文：基于 FlashAttention 技术，我们将基座模型的上下文长度（Context Length）由 ChatGLM-6B 的 2K 扩展到了 32K，并在对话阶段使用 8K 的上下文长度训练，允许更多轮次的对话。但当前版本的 ChatGLM2-6B 对单轮超长文档的理解能力有限，我们会在后续迭代升级中着重进行优化。更高效的推理：基于 Multi-Query Attention 技术，ChatGLM2-6B 有更高效的推理速度和更低的显存占用：在官方的模型实现下，推理速度相比初代提升了 42%，INT4 量化下，6G 显存支持的对话长度由 1K 提升到了 8K。更开放的协议：ChatGLM2-6B 权重对学术研究完全开放，在获得官方的书面许可后，亦允许商业使用。如果您发现我们的开源模型对您的业务有用，我们欢迎您对下一代模型 ChatGLM3 研发的捐赠。论文名称：ChatGLM2-6B: An Open Bilingual Chat LLM | 开源双语对话语言模型论文地址：Github 代码：https://github.com/THUDM/ChatGLM2-6B动机：在主要评估LLM模型中文能力的 C-Eval 榜单中，截至6月25日 ChatGLM2 模型以 71.1 的分数位居 Rank 0 ，ChatGLM2-6B 模型以 51.7 的分数位居 Rank 6，是榜单上排名最高的开源模型。介绍：ChatGLM2-6B 是开源中英双语对话模型 ChatGLM-6B 的第二代版本，在保留了初代模型对话流畅、部署门槛较低等众多优秀特性的基础之上，ChatGLM2-6B 引入了如下新特性：【关于 ChatGLM2 + LoRA 进行finetune 】那些你不知道的事论文名称：ChatGLM2-6B: An Open Bilingual Chat LLM | 开源双语对话语言模型论文地址：Github 代码：https://github.com/THUDM/ChatGLM2-6B介绍：本教程主要介绍对于 ChatGLM2-6B 模型基于 LoRA 进行finetune。【LLMs 入门实战 】基于 🤗PEFT 的高效 🤖ChatGLM2-6B 微调ChatGLM2-6B Freeze 微调：Fine-tuning the MLPs in the last n blocks of the model.ChatGLM2-6B P-Tuning V2 微调：Fine-tuning the prefix encoder of the model.ChatGLM2-6B LoRA 微调：Fine-tuning the low-rank adapters of the model.微调方式：【LLMs 入门实战】基于 🤗QLoRA 的高效 🤖ChatGLM2-6B 微调介绍：本项目使用 https://github.com/huggingface/peft 库，实现了 ChatGLM2-6B 模型4bit的 QLoRA 高效微调，可以在一张RTX3060上完成全部微调过程。第十一重 Llama2【LLMs 入门实战】 Llama2 模型学习与实战官网：https://ai.meta.com/llama/论文名称：《Llama 2: Open Foundation and Fine-Tuned Chat Models》论文地址：https://ai.meta.com/research/publications/llama-2-open-foundation-and-fine-tuned-chat-models/演示平台：https://llama2.ai/Github 代码：https://github.com/facebookresearch/llama模型下载地址：https://ai.meta.com/resources/models-and-libraries/llama-downloads/介绍：此次 Meta 发布的 Llama 2 模型系列包含 70 亿、130 亿和 700 亿三种参数变体。此外还训练了 340 亿参数变体，但并没有发布，只在技术报告中提到了。据介绍，相比于 Llama 1，Llama 2 的训练数据多了 40%，上下文长度也翻倍，并采用了分组查询注意力机制。具体来说，Llama 2 预训练模型是在 2 万亿的 token 上训练的，精调 Chat 模型是在 100 万人类标记数据上训练的。【LLMs 入门实战】Chinese-Llama-2-7b 模型学习与实战https://huggingface.co/ziqingyang/chinese-llama-2-7bhttps://huggingface.co/LinkSoul/Chinese-Llama-2-7b-4bit官网：https://ai.meta.com/llama/论文名称：《Llama 2: Open Foundation and Fine-Tuned Chat Models》论文地址：https://ai.meta.com/research/publications/llama-2-open-foundation-and-fine-tuned-chat-models/演示平台：https://huggingface.co/spaces/LinkSoul/Chinese-Llama-2-7bGithub 代码：https://github.com/LinkSoul-AI/Chinese-Llama-2-7b模型下载地址：介绍：自打 LLama-2 发布后就一直在等大佬们发布 LLama-2 的适配中文版，也是这几天蹲到了一版由 LinkSoul 发布的 Chinese-Llama-2-7b，其共发布了一个常规版本和一个 4-bit 的量化版本，今天我们主要体验下 Llama-2 的中文逻辑顺便看下其训练样本的样式，后续有机会把训练和微调跑起来。第十重 BaichuanBaichuan-13B【LLMs 入门实战 】 Baichuan-13B 模型学习与实战更大尺寸、更多数据：Baichuan-13B 在 Baichuan-7B 的基础上进一步扩大参数量到 130 亿，并且在高质量的语料上训练了 1.4 万亿 tokens，超过 LLaMA-13B 40%，是当前开源 13B 尺寸下训练数据量最多的模型。支持中英双语，使用 ALiBi 位置编码，上下文窗口长度为 4096。同时开源预训练和对齐模型：预训练模型是适用开发者的“基座”，而广大普通用户对有对话功能的对齐模型具有更强的需求。因此本次开源同时发布了对齐模型（Baichuan-13B-Chat），具有很强的对话能力，开箱即用，几行代码即可简单的部署。更高效的推理：为了支持更广大用户的使用，本次同时开源了 int8 和 int4 的量化版本，相对非量化版本在几乎没有效果损失的情况下大大降低了部署的机器资源门槛，可以部署在如 Nvidia 3090 这样的消费级显卡上。开源免费可商用：Baichuan-13B 不仅对学术研究完全开放，开发者也仅需邮件申请并获得官方商用许可后，即可以免费商用。官方微调过（指令对齐）:https://huggingface.co/baichuan-inc/Baichuan-13B-Chat预训练大模型（未经过微调）:https://huggingface.co/baichuan-inc/Baichuan-13B-Basebaichuan-inc/Baichuan-13B：https://github.com/baichuan-inc/Baichuan-13BBaichuan-13B 大模型：介绍：Baichuan-13B 是由百川智能继 Baichuan-7B 之后开发的包含 130 亿参数的开源可商用的大规模语言模型，在权威的中文和英文 benchmark 上均取得同尺寸最好的效果。Baichuan-13B 有如下几个特点：baichuan-7B【LLMs 入门实战 】 baichuan-7B 学习与实战论文名称：论文地址：Github 代码： https://github.com/baichuan-inc/baichuan-7B模型：介绍：由百川智能开发的一个开源可商用的大规模预训练语言模型。基于Transformer结构，在大约1.2万亿tokens上训练的70亿参数模型，支持中英双语，上下文窗口长度为4096。在标准的中文和英文权威benchmark（C-EVAL/MMLU）上均取得同尺寸最好的效果。第二重 Stanford Alpaca 7B【LLMs 入门实战 —— 五 】Stanford Alpaca 7B 模型学习与实战介绍：本教程提供了对LLaMA模型进行微调的廉价亲民 LLMs 学习和微调 方式，主要介绍对于 Stanford Alpaca 7B 模型在特定任务上 的 微调实验，所用的数据为OpenAI提供的GPT模型API生成质量较高的指令数据（仅52k）。第三重 Chinese-LLaMA-Alpaca【LLMs 入门实战 —— 六 】Chinese-LLaMA-Alpaca 模型学习与实战介绍：本教程主要介绍了 Chinese-ChatLLaMA,提供中文对话模型 ChatLLama 、中文基础模型 LLaMA-zh 及其训练数据。模型基于 TencentPretrain 多模态预训练框架构建第四重 小羊驼 Vicuna【LLMs 入门实战 —— 七 】小羊驼 Vicuna模型学习与实战介绍：UC伯克利学者联手CMU、斯坦福等，再次推出一个全新模型70亿/130亿参数的Vicuna，俗称「小羊驼」（骆马）。小羊驼号称能达到GPT-4的90%性能第五重 MOSS【LLMs 入门实战 —— 十三 】MOSS 模型学习与实战介绍：MOSS是一个支持中英双语和多种插件的开源对话语言模型，moss-moon系列模型具有160亿参数，在FP16精度下可在单张A100/A800或两张3090显卡运行，在INT4/8精度下可在单张3090显卡运行。MOSS基座语言模型在约七千亿中英文以及代码单词上预训练得到，后续经过对话指令微调、插件增强学习和人类偏好训练具备多轮对话能力及使用多种插件的能力。局限性：由于模型参数量较小和自回归生成范式，MOSS仍然可能生成包含事实性错误的误导性回复或包含偏见/歧视的有害内容，请谨慎鉴别和使用MOSS生成的内容，请勿将MOSS生成的有害内容传播至互联网。若产生不良后果，由传播者自负。第六重 BLOOMz【LLMs 入门实战 —— 十四 】 BLOOMz 模型学习与实战介绍：大型语言模型（LLMs）已被证明能够根据一些演示或自然语言指令执行新的任务。虽然这些能力已经导致了广泛的采用，但大多数LLM是由资源丰富的组织开发的，而且经常不对公众开放。作为使这一强大技术民主化的一步，我们提出了BLOOM，一个176B参数的开放性语言模型，它的设计和建立要感谢数百名研究人员的合作。BLOOM是一个仅有解码器的Transformer语言模型，它是在ROOTS语料库上训练出来的，该数据集包括46种自然语言和13种编程语言（共59种）的数百个来源。我们发现，BLOOM在各种基准上取得了有竞争力的性能，在经历了多任务提示的微调后，其结果更加强大。模型地址：https://huggingface.co/bigscience/bloomz第七重 BELLE【LLMs 入门实战 —— 十五 】 ",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			DeduplicateId:      "123",
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&origin,
	})
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	emb, err := CalculateEmbedding(origin.Post.Title + origin.Post.Content)
	t.Log(emb)
	assert.Nil(t, err)
	assert.Len(t, emb, 100)
	msgs, _ := reader.ReceiveMessages(1)

	msg, err := processor.ProcessOneCralwerMessage(msgs[0])
	assert.Nil(t, err)
	assert.NotNil(t, msg)
}

func TestDecodeCrawlerMessage(t *testing.T) {
	db, _ := CreateTempDB(t)

	origin := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				Id: "2",
			},
			Title:              "hello",
			Content:            "hello world!",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&origin,
	})

	// Inject test dependent reader
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})

	msgs, _ := reader.ReceiveMessages(1)
	assert.Equal(t, len(msgs), 1)

	// This is the function we tested here
	//given MessageQueueMessage, decode it into struct
	decodedObj, _ := processor.DecodeCrawlerMessage(msgs[0])

	assert.True(t, cmp.Equal(*decodedObj, origin, cmpopts.IgnoreUnexported(
		protocol.CrawlerMessage{},
		protocol.CrawlerMessage_CrawledPost{},
		protocol.CrawledSubSource{},
		timestamppb.Timestamp{},
	)))
}

func PrepareTestDBClient(db *gorm.DB) *client.Client {
	client := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB:          db,
		SignalChans: nil,
	}})))
	return client
}

func TestProcessCrawlerMessage(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)

	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	sourceId2 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId1, false, db, client)
	subSourceId2 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api_2", "test_externalid", sourceId2, false, db, client)

	feedId, _, columnId := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	feedId2, _, columnId2 := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api_2", DataExpressionJsonForTest, []string{subSourceId1, subSourceId2}, model.VisibilityPrivate, db, client)
	TestUserSubscribeColumnAndValidate(t, uid, columnId, db, client)
	TestUserSubscribeColumnAndValidate(t, uid, columnId2, db, client)

	testTimeStamp := timestamppb.Now()

	msgToTwoFeeds := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				Name:     "test_subsource_for_feeds_api",
				SourceId: sourceId1,
			},
			Title:              "msgToTwoFeeds",
			Content:            "老王做空以太坊",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"Tesla", "中概股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: testTimeStamp,
		},
		CrawledAt:      testTimeStamp,
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	msgToOneFeed := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "2",
			SubSource: &protocol.CrawledSubSource{
				Name:     "test_subsource_for_feeds_api_2",
				SourceId: sourceId2,
			},
			Title:              "msgToOneFeed",
			Content:            "老王做空以太坊_2",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: testTimeStamp,
		},
		CrawledAt:      testTimeStamp,
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	var msgDataExpressionUnMatched protocol.CrawlerMessage
	copier.Copy(&msgDataExpressionUnMatched, &msgToOneFeed)
	msgDataExpressionUnMatched.Post.Title = "msgDataExpressionUnMatched"
	msgDataExpressionUnMatched.Post.Content = "马斯克做空以太坊"
	msgDataExpressionUnMatched.Post.DeduplicateId = "3"

	t.Run("Test Publish Post to Feed based on subsource", func(t *testing.T) {
		// msgToTwoFeeds is from subsource 1 which in 2 feeds
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToTwoFeeds,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToTwoFeeds.Post.Title)
		require.Equal(t, 2, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, feedId2, post.PublishedFeeds[1].Id)
		require.Equal(t, testTimeStamp.Seconds, post.ContentGeneratedAt.Unix())
		require.Equal(t, testTimeStamp.Seconds, post.CrawledAt.Unix())
		require.Equal(t, "Tesla,中概股", post.Tag)
	})

	t.Run("Test Publish Post to Feed based on source", func(t *testing.T) {
		// msgToOneFeed is from source 1 which in 1 feed
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToOneFeed.Post.Title)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, post.PublishedFeeds[0].Id, feedId2)
		require.Equal(t, 2, len(post.ImageUrls))
		require.Equal(t, "1", post.ImageUrls[0])
		require.Equal(t, 2, len(post.FileUrls))
		require.Equal(t, "aaa", post.OriginUrl)
		require.Equal(t, "电动车,港股", post.Tag)
	})

	t.Run("Test Post deduplication", func(t *testing.T) {
		// send message again
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing Again, there should be no new post
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.NoError(t, err)

		var count int64
		processor.DB.Model(&model.Post{}).Where("title = ?", msgToOneFeed.Post.Title).Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("Test Publish Post to Feed based on source Data Expression not matched", func(t *testing.T) {
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgDataExpressionUnMatched,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgDataExpressionUnMatched.Post.Title)
		require.Equal(t, 0, len(post.PublishedFeeds))
	})
}

func TestProcessCrawlerRetweetMessage(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_1", "test_externalid", sourceId1, false, db, client)
	feedId, _, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)

	msgToOneFeed := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好", // This doesn't match data exp
			Content:            "老王干得好",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "2",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "test_subsource_2",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊", // This matches data exp
				Content:            "老王做空以太坊详情",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	t.Run("Test publish post with retweet sharing", func(t *testing.T) {
		// msgToTwoFeeds is from subsource 1 which in 2 feeds
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload(clause.Associations).First(&post, "title = ?", msgToOneFeed.Post.Title)

		require.Equal(t, msgToOneFeed.Post.Title, post.Title)
		require.Equal(t, msgToOneFeed.Post.Content, post.Content)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, "电动车,港股", post.Tag)

		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.Title, post.SharedFromPost.Title)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.Content, post.SharedFromPost.Content)
		require.Equal(t, true, post.SharedFromPost.InSharingChain)
		require.Equal(t, 0, len(post.SharedFromPost.PublishedFeeds))
		require.Equal(t, "Tesla,中概股", post.SharedFromPost.Tag)

		// Check isFromSharedPost mark is set correctly
		var subScourceOrigin model.SubSource
		processor.DB.Preload(clause.Associations).Where("id=?", post.SubSourceID).First(&subScourceOrigin)
		require.False(t, subScourceOrigin.IsFromSharedPost)

		// Check new subsource is created
		var subScourceShared model.SubSource
		processor.DB.Preload(clause.Associations).Where("id=?", post.SharedFromPost.SubSourceID).First(&subScourceShared)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.Name, subScourceShared.Name)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.ExternalId, subScourceShared.ExternalIdentifier)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.OriginUrl, subScourceShared.OriginUrl)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.AvatarUrl, subScourceShared.AvatarUrl)
		require.True(t, subScourceShared.IsFromSharedPost)
	})
}

func TestRetweetMessageProcessSubsourceCreation(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)

	msgOne := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好", // This doesn't match data exp
			Content:            "老王干得好",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "2",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "test_subsource_2",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊", // This matches data exp
				Content:            "老王做空以太坊详情",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgOne,
	})
	msgs, _ := reader.ReceiveMessages(1)
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err := processor.ProcessOneCralwerMessage(msgs[0])
	require.Nil(t, err)
	var subScourceOne model.SubSource
	var subScourceTwo model.SubSource
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_1").First(&subScourceOne)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_2").First(&subScourceTwo)
	require.False(t, subScourceOne.IsFromSharedPost)
	require.True(t, subScourceTwo.IsFromSharedPost)

	msgTwo := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "3",
			SubSource: &protocol.CrawledSubSource{
				// Changing order of the two subsources
				Name:       "test_subsource_2",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好_new_msg", //avoid dedup error
			Content:            "老王干得好_new_msg",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "4",
				SubSource: &protocol.CrawledSubSource{
					// Changing order of the two subsources
					Name:       "test_subsource_1",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊_new_msg", //avoid dedup error
				Content:            "老王做空以太坊详情_new_msg",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader = NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgTwo,
	})
	msgs, _ = reader.ReceiveMessages(1)
	processor = NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err = processor.ProcessOneCralwerMessage(msgs[0])
	require.Nil(t, err)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_1").First(&subScourceOne)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_2").First(&subScourceTwo)
	require.False(t, subScourceOne.IsFromSharedPost)
	require.False(t, subScourceTwo.IsFromSharedPost)
}

func TestMessagePublishToManyFeeds(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_1", "test_externalid", sourceId1, false, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)

	msgOne := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王做空以太坊", // This matches data exp
			Content:            "老王做空以太坊",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgOne,
	})
	msgs, _ := reader.ReceiveMessages(1)
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err := processor.ProcessOneCralwerMessage(msgs[0])
	require.NoError(t, err)
	var post model.Post
	processor.DB.Preload(clause.Associations).Where("content=?", "老王做空以太坊").First(&post)
	require.Equal(t, 10, len(post.PublishedFeeds))
	require.NotEqual(t, post.PublishedFeeds[1].Id, post.PublishedFeeds[0].Id)
}

func TestPublishThread(t *testing.T) {
	db, name := CreateTempDB(t)
	fmt.Println(name)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)

	msgOne := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "last",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "Last Post SS name",
				SourceId:   sourceId,
				ExternalId: "last",
				AvatarUrl:  "last",
				OriginUrl:  "last",
			},
			Title:              "last", // This matches data exp
			Content:            "last",
			ImageUrls:          []string{"last_1", "last_2"},
			OriginUrl:          "last",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "shared_by_last",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "Shared by Last Post SS name",
					SourceId:   sourceId,
					ExternalId: "shared_by_last",
					AvatarUrl:  "shared_by_last",
					OriginUrl:  "shared_by_last",
				},
				Title:              "shared by last", // This matches data exp
				Content:            "shared by last",
				ImageUrls:          []string{"shared_by_last_1", "shared_by_last_2"},
				OriginUrl:          "shared by last",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
			ReplyTo: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "second",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "second ss name",
					SourceId:   sourceId,
					ExternalId: "second",
					AvatarUrl:  "second",
					OriginUrl:  "second",
				},
				Title:              "second", // This matches data exp
				Content:            "second",
				ImageUrls:          []string{"second_1", "second_2"},
				OriginUrl:          "second",
				ContentGeneratedAt: timestamppb.Now(),
				SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
					DeduplicateId: "shared_by_second",
					SubSource: &protocol.CrawledSubSource{
						// New subsource to be created and mark as isFromSharedPost
						Name:       "shared by second ss name",
						SourceId:   sourceId,
						ExternalId: "shared by second",
						AvatarUrl:  "shared by second",
						OriginUrl:  "shared by second",
					},
					Title:              "shared by second", // This matches data exp
					Content:            "shared by second",
					ImageUrls:          []string{"shared_by_second_1", "shared_by_second_2"},
					OriginUrl:          "shared by second",
					ContentGeneratedAt: &timestamppb.Timestamp{},
				},
				ReplyTo: &protocol.CrawlerMessage_CrawledPost{
					DeduplicateId: "first",
					SubSource: &protocol.CrawledSubSource{
						// New subsource to be created and mark as isFromSharedPost
						Name:       "first ss name",
						SourceId:   sourceId,
						ExternalId: "first",
						AvatarUrl:  "first",
						OriginUrl:  "first",
					},
					Title:              "first", // This matches data exp
					Content:            "first",
					ImageUrls:          []string{"first_1", "first_2"},
					OriginUrl:          "first",
					ContentGeneratedAt: &timestamppb.Timestamp{},
					SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
						DeduplicateId: "shared_by_first",
						SubSource: &protocol.CrawledSubSource{
							// New subsource to be created and mark as isFromSharedPost
							Name:       "shared by first ss name",
							SourceId:   sourceId,
							ExternalId: "shared by first",
							AvatarUrl:  "shared by first",
							OriginUrl:  "shared by first",
						},
						Title:              "shared by first", // This matches data exp
						Content:            "shared by first",
						ImageUrls:          []string{"shared_by_first_1", "shared_by_first_2"},
						OriginUrl:          "shared by first",
						ContentGeneratedAt: &timestamppb.Timestamp{},
					},
				},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgOne,
	})
	msgs, _ := reader.ReceiveMessages(1)
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err := processor.ProcessOneCralwerMessage(msgs[0])
	require.Nil(t, err)
	var post model.Post
	processor.DB.
		Preload(clause.Associations).
		Preload("SharedFromPost.SubSource").
		// Maintain a chronological order of reply thread.
		Preload("ReplyThread", func(db *gorm.DB) *gorm.DB {
			return db.Order("posts.created_at ASC")
		}).
		Preload("ReplyThread.SubSource").
		Preload("ReplyThread.SharedFromPost").
		Preload("ReplyThread.SharedFromPost.SubSource").
		Where("deduplicate_id=?", "last").
		First(&post)

	require.Equal(t, len(post.ReplyThread), 2)
	require.Equal(t, post.Content, "last")
	require.Equal(t, post.SharedFromPost.Content, "shared by last")
	require.Equal(t, post.ReplyThread[0].Content, "second")
	require.Equal(t, post.ReplyThread[0].SharedFromPost.Content, "shared by second")
	require.Equal(t, len(post.ReplyThread[0].ReplyThread), 0)
	require.Equal(t, post.ReplyThread[1].Content, "first")
	require.Equal(t, post.ReplyThread[1].SharedFromPost.Content, "shared by first")
	require.Equal(t, len(post.ReplyThread[1].ReplyThread), 0)

	var subSources []model.SubSource
	result := processor.DB.Find(&subSources)
	// 6 created by processing message, with a default created by CreateSource API
	require.Equal(t, result.RowsAffected, int64(7))
}
