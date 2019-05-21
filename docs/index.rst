.. KubeEdge documentation master file, created by
   sphinx-quickstart on Fri Feb  8 12:12:47 2019.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to KubeEdge's documentation!
====================================


.. figure:: images/KubeEdge_logo.png
      :width: 150px
      :align: right
      :target: https://kubeedge.io

KubeEdge is an open source system for extending native containerized 
application orchestration capabilities to hosts at Edge.

.. toctree::
   KubeEdge Home <https://kubeedge.io>

.. toctree::
   :maxdepth: 2
   :caption: Getting Started
   
   getting-started/getting-started
   getting-started/contribute.md
   getting-started/roadmap.md
   getting-started/support.md
   getting-started/community-membership
   getting-started/release_package
   getting-started/reporting_bugs

.. toctree::
   :maxdepth: 2
   :caption: General Concepts

   modules/kubeedge.md
   modules/beehive

.. toctree::
   :maxdepth: 2
   :caption: Edge Concepts

   modules/edge/edged
   modules/edge/eventbus
   modules/edge/metamanager
   modules/edge/edgehub
   modules/edge/devicetwin

.. toctree::
   :maxdepth: 2
   :caption: Cloud Concepts

   modules/cloud/controller
   modules/cloud/cloudhub
   modules/cloud/device_controller

.. toctree::
   :maxdepth: 2
   :caption: Edgesite

   modules/edgesite

.. toctree::
   :maxdepth: 2
   :caption: Mappers

   mappers/bluetooth_mapper
   mappers/modbus_mapper

.. toctree::
   :maxdepth: 2
   :caption: Setup

   setup/requirements
   setup/setup
   One Click Installer <setup/installer_setup>
   setup/cross-compilation
   setup/memfootprint-test-setup
   Integrate with HuaweiCloud [Intelligent EdgeFabric (IEF)] <guides/try_kubeedge_with_ief>


.. toctree::
   :maxdepth: 2
   :caption: Guides

   guides/message_topics
   guides/unit_test_guide
   guides/device_crd_guide
   guides/edgemesh_test_env_guide

.. toctree::
   :maxdepth: 2
   :caption: Troubleshooting

   troubleshooting/troubleshooting




