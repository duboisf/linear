FROM ubuntu:22.04

ARG TZ
ENV TZ="$TZ"
ENV DEBIAN_FRONTEND=noninteractive

# Install basic development tools, iptables/ipset, and prerequisites for git PPA
RUN apt-get update && apt-get install -y --no-install-recommends \
  software-properties-common \
  curl \
  ca-certificates \
  wget \
  less \
  procps \
  sudo \
  zsh \
  man-db \
  unzip \
  gnupg2 \
  iptables \
  ipset \
  iproute2 \
  dnsutils \
  aggregate \
  jq \
  nano \
  vim \
  locales \
  && locale-gen en_US.UTF-8 \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8

# Add Git PPA and install latest git
RUN add-apt-repository ppa:git-core/ppa && \
  apt-get update && \
  apt-get install -y git && \
  apt-get clean && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
  chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
  apt-get update && \
  apt-get install -y gh && \
  apt-get clean && rm -rf /var/lib/apt/lists/*

# Create dev user with matching UID/GID from host
ARG UID=1000
ARG GID=1000
RUN groupadd -g ${GID} dev && \
    useradd -m -s /bin/zsh -u ${UID} -g ${GID} -G sudo dev && \
    echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Ensure dev user has access to /usr/local/share
RUN mkdir -p /usr/local/share/npm-global && \
  chown -R dev:dev /usr/local/share

ARG USERNAME=dev

# Persist bash history
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  && mkdir /commandhistory \
  && touch /commandhistory/.bash_history \
  && chown -R $USERNAME /commandhistory

# Set `DEVCONTAINER` environment variable to help with orientation
ENV DEVCONTAINER=true

# Create workspace and config directories and set permissions
RUN mkdir -p /workspace /home/dev/.claude /home/dev/.codex && \
  chown -R dev:dev /workspace /home/dev/.claude /home/dev/.codex

WORKDIR /workspace

ARG GIT_DELTA_VERSION=0.18.2
RUN ARCH=$(dpkg --print-architecture) && \
  wget "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"

# Set up non-root user
USER dev

# Set up mise for dev user
ENV MISE_DATA_DIR=/home/dev/.local/share/mise
ENV MISE_CONFIG_DIR=/home/dev/.config/mise
ENV PATH="/home/dev/.local/bin:$PATH"

RUN curl https://mise.run | sh && \
  echo 'eval "$(~/.local/bin/mise activate zsh)"' >> ~/.zshrc && \
  echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc

# Install Node.js and fzf via mise
RUN /home/dev/.local/bin/mise use --global node@20 fzf

# Set the default shell to zsh rather than sh
ENV SHELL=/bin/zsh

# Set the default editor and visual
ENV EDITOR=nano
ENV VISUAL=nano

# Default powerline10k theme
ARG ZSH_IN_DOCKER_VERSION=1.2.0
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
  -p git \
  -a 'eval "$(~/.local/bin/mise activate zsh)"' \
  -a 'eval "$(fzf --zsh)"' \
  -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  -x

# Install native Claude Code
RUN curl -fsSL https://claude.ai/install.sh | bash

# Install OpenAI Codex CLI via mise-managed npm
RUN /home/dev/.local/bin/mise exec -- npm install -g @openai/codex

# Copy and set up firewall script
COPY init-firewall.sh /usr/local/bin/
USER root
RUN chmod +x /usr/local/bin/init-firewall.sh && \
  echo "dev ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/dev-firewall && \
  chmod 0440 /etc/sudoers.d/dev-firewall
USER dev
