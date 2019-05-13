FROM ubuntu:16.04
RUN echo '#!/bin/bash' > /infinite.sh
RUN echo 'while /bin/true' >> /infinite.sh
RUN echo 'do' >> /infinite.sh
RUN echo ' sleep 5' >> /infinite.sh
RUN echo 'done' >> /infinite.sh
RUN chmod +x /infinite.sh
CMD ["./infinite.sh"]
